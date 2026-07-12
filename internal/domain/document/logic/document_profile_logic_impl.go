package logic

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// ProfileLogicImpl 文档画像逻辑实现
type ProfileLogicImpl struct {
	repo adapter.DocumentRepository
	port *adapter.DocumentPort
}

var _ ProfileLogic = (*ProfileLogicImpl)(nil)

// NewProfileLogicImpl 构造函数
func NewProfileLogicImpl(repo adapter.DocumentRepository, port *adapter.DocumentPort) *ProfileLogicImpl {
	return &ProfileLogicImpl{repo: repo, port: port}
}

// 画像状态常量
const (
	profileStatusSuccess = 2
	profileSourceAuto    = "auto"
)

// 文档类型常量
const (
	docTypeFAQ          = "faq"
	docTypeTroubleshoot = "troubleshooting"
	docTypeRule         = "rule"
	docTypeSpec         = "spec"
	docTypeManual       = "manual"
	docTypeIntro        = "intro"
)

// knowledgeScopeCode 知识范围编码
const (
	scopeOperationRule = "operation_rule"
	scopeRobotStrategy = "robot_strategy"
	scopeDeployment    = "deployment"
	scopeTroubleshoot  = "troubleshooting"
	scopeProduct       = "product"
	scopeGeneral       = "general_document"
)

// 章节编码/序号剥离正则
var (
	sectionCodePrefixRegexp = regexp.MustCompile(`^(第[一二三四五六七八九十百0-9]+[章节条部分]\s*)|(\d+(?:\.\d+)+\s*)`)
	whitespaceRegexp        = regexp.MustCompile(`\s+`)
)

// ==================== 对外接口 ====================

// GenerateProfile 根据分析结果生成/更新文档画像
func (p *ProfileLogicImpl) GenerateProfile(ctx context.Context, documentId int64, analysisResult *vo.DocumentAnalysisResult, structureNodes []*entity.DocumentStructureNode) (*entity.DocumentProfile, error) {
	if documentId == 0 {
		return nil, errors.New("documentId 不能为空")
	}
	document, err := p.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}

	parsedText := ""
	if analysisResult != nil {
		parsedText = strutil.Trim(analysisResult.ParsedText)
	}

	draft := p.buildDraft(document, parsedText, structureNodes)

	existing, err := p.repo.SelectProfileByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}

	profile := &entity.DocumentProfile{
		DocumentId:           documentId,
		DocumentSummary:      draft.documentSummary,
		DocumentType:         draft.documentType,
		CoreTopics:           utils.ToCompactJSON(draft.coreTopics),
		ExampleQuestions:     utils.ToCompactJSON(draft.exampleQuestions),
		GraphFriendly:        boolToInt(draft.graphFriendly),
		SupportsGraphOutline: boolToInt(draft.supportsGraphOutline),
		SupportsItemLookup:   boolToInt(draft.supportsItemLookup),
		SupportsGraphAssist:  boolToInt(draft.supportsGraphAssist),
		ProfileSource:        profileSourceAuto,
		ProfileStatus:        profileStatusSuccess,
	}
	if existing.ID == 0 {
		profile.ID = utils.GetSnowflakeNextID()
		profile.ProfileVersion = 1
	} else {
		profile.ID = existing.ID
		profile.ProfileVersion = existing.ProfileVersion + 1
	}

	if err = p.repo.UpsertProfile(ctx, profile); err != nil {
		return nil, err
	}

	if err = p.repo.UpdateDocumentById(ctx, document); err != nil {
		logx.Errorf("backfill document metadata failed: documentId=%d, err=%v", documentId, err)
	}
	logx.Infof("文档画像生成完成: documentId=%d, documentType=%s, graphFriendly=%v, supportsItemLookup=%v, scopeCode=%s, businessCategory=%s, tags=%s",
		documentId, draft.documentType, draft.graphFriendly, draft.supportsItemLookup, draft.knowledgeScopeCode, draft.businessCategory, draft.documentTags)
	return profile, nil
}

// GetProfileByDocumentId 根据文档ID获取画像
func (p *ProfileLogicImpl) GetProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	if documentId == 0 {
		return nil, nil
	}
	return p.repo.SelectProfileByDocumentId(ctx, documentId)
}

// RegenerateProfile 重新生成文档画像
func (p *ProfileLogicImpl) RegenerateProfile(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	if documentId == 0 {
		return nil, errors.New("documentId 不能为空")
	}
	document, err := p.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, errors.New("文档不存在")
	}
	var parsedText string
	if strutil.IsNotBlank(document.ParseTextPath) && p.port != nil {
		if t, e := p.port.DownloadText(ctx, document.ParseTextPath); e == nil {
			parsedText = t
		}
	}
	structureNodes, err := p.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	analysisResult := &vo.DocumentAnalysisResult{ParsedText: parsedText}
	return p.GenerateProfile(ctx, documentId, analysisResult, structureNodes)
}

// BatchRegenerateProfiles 批量重新生成文档画像
func (p *ProfileLogicImpl) BatchRegenerateProfiles(ctx context.Context, documentIds []int64) ([]*entity.DocumentProfile, error) {
	if len(documentIds) == 0 {
		return []*entity.DocumentProfile{}, nil
	}
	result := make([]*entity.DocumentProfile, 0, len(documentIds))
	for _, id := range documentIds {
		if id == 0 {
			continue
		}
		profile, err := p.RegenerateProfile(ctx, id)
		if err != nil {
			return result, err
		}
		result = append(result, profile)
	}
	return result, nil
}

// ==================== 内部实现：构建画像草稿 ====================

// profileDraft 画像草稿
type profileDraft struct {
	documentSummary      string
	documentType         string
	coreTopics           []string
	exampleQuestions     []string
	graphFriendly        bool
	supportsGraphOutline bool
	supportsItemLookup   bool
	supportsGraphAssist  bool
	knowledgeScopeCode   string
	knowledgeScopeName   string
	businessCategory     string
	documentTags         string
}

func (p *ProfileLogicImpl) buildDraft(document *entity.Document, parsedText string, structureNodes []*entity.DocumentStructureNode) *profileDraft {
	sectionTitles := p.extractSectionTitles(structureNodes)
	supportsItemLookup := false
	for _, node := range structureNodes {
		if node == nil {
			continue
		}
		if node.NodeType == vo.NodeTypeStep || node.NodeType == vo.NodeTypeListItem {
			supportsItemLookup = true
			break
		}
	}
	supportsGraphOutline := len(sectionTitles) >= 2
	graphFriendly := supportsItemLookup || supportsGraphOutline
	combined := combinedText(document, parsedText, sectionTitles)
	documentType := vo.InferDocumentType(combined, supportsItemLookup)
	coreTopics := p.buildCoreTopics(document, sectionTitles)
	exampleQuestions := p.buildExampleQuestions(documentType, coreTopics)
	summary := p.buildSummary(document, sectionTitles, parsedText)
	knowledgeScopeCode := p.inferKnowledgeScopeCode(document, sectionTitles, parsedText)
	knowledgeScopeName := p.inferKnowledgeScopeName(knowledgeScopeCode)
	businessCategory := p.inferBusinessCategory(documentType, parsedText)
	documentTags := p.buildDocumentTags(document, knowledgeScopeCode, documentType, coreTopics)
	return &profileDraft{
		documentSummary:      summary,
		documentType:         documentType,
		coreTopics:           coreTopics,
		exampleQuestions:     exampleQuestions,
		graphFriendly:        graphFriendly,
		supportsGraphOutline: supportsGraphOutline,
		supportsItemLookup:   supportsItemLookup,
		supportsGraphAssist:  true,
		knowledgeScopeCode:   knowledgeScopeCode,
		knowledgeScopeName:   knowledgeScopeName,
		businessCategory:     businessCategory,
		documentTags:         documentTags,
	}
}

// backfillDocumentMetadata 将草稿中的关键字段回填到文档主表（仅在原字段为空时）
func (p *ProfileLogicImpl) backfillDocumentMetadata(document *entity.Document, draft *profileDraft) error {
	if document == nil || draft == nil {
		return nil
	}
	changed := false
	if strutil.IsBlank(document.KnowledgeScopeCode) && strutil.IsNotBlank(draft.knowledgeScopeCode) {
		document.KnowledgeScopeCode = draft.knowledgeScopeCode
		changed = true
	}
	if strutil.IsBlank(document.KnowledgeScopeName) && strutil.IsNotBlank(draft.knowledgeScopeName) {
		document.KnowledgeScopeName = draft.knowledgeScopeName
		changed = true
	}
	if strutil.IsBlank(document.BusinessCategory) && strutil.IsNotBlank(draft.businessCategory) {
		document.BusinessCategory = draft.businessCategory
		changed = true
	}
	if strutil.IsBlank(document.DocumentTags) && strutil.IsNotBlank(draft.documentTags) {
		document.DocumentTags = draft.documentTags
		changed = true
	}
	if !changed {
		return nil
	}
	return p.repo.UpdateDocumentById(ctx, document)
}

// extractSectionTitles 提取章节标题（去重、取前 8 条）
func (p *ProfileLogicImpl) extractSectionTitles(structureNodes []*entity.DocumentStructureNode) []string {
	if len(structureNodes) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	result := make([]string, 0, 8)
	for _, node := range structureNodes {
		if node == nil || node.NodeType != vo.NodeTypeSection {
			continue
		}
		title := strutil.Trim(node.Title)
		_, ok := seen[title]
		if title != "" && !ok {
			seen[title] = struct{}{}
			result = append(result, title)
		}
		if len(result) >= 8 {
			break
		}
	}
	return result
}

// inferDocumentType 推断文档类型
func (p *ProfileLogicImpl) inferDocumentType(document *entity.Document, parsedText string, sectionTitles []string, supportsItemLookup bool) string {
	combined := combinedText(document, parsedText, sectionTitles)
	if strings.Contains(combined, "faq") || strings.Contains(combined, "常见问题") {
		return docTypeFAQ
	}
	if strings.Contains(combined, "故障") || strings.Contains(combined, "排查") || strings.Contains(combined, "检查顺序") {
		return docTypeTroubleshoot
	}
	if strings.Contains(combined, "规则") || strings.Contains(combined, "制度") {
		return docTypeRule
	}
	if strings.Contains(combined, "规格") || strings.Contains(combined, "参数") {
		return docTypeSpec
	}
	if supportsItemLookup || strings.Contains(combined, "手册") || strings.Contains(combined, "指南") || strings.Contains(combined, "部署") {
		return docTypeManual
	}
	return docTypeIntro
}

// buildCoreTopics 构建核心话题
func (p *ProfileLogicImpl) buildCoreTopics(document *entity.Document, sectionTitles []string) []string {
	fileName := strutil.Trim(document.DocumentName)
	fileTopic := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	sectionTitles = append(sectionTitles, fileTopic)
	result := utils.DistinctAndLimit(sectionTitles, 6, func(title string) string {
		return stripSectionCode(title)
	})
	return result
}

// buildExampleQuestions 构造示例问题（根据文档类型不同添加后缀）
func (p *ProfileLogicImpl) buildExampleQuestions(documentType string, coreTopics []string) []string {
	result := make([]string, 0, len(coreTopics))
	seen := make(map[string]struct{})
	for _, topic := range coreTopics {
		var q string
		switch documentType {
		case docTypeTroubleshoot:
			q = topic + "的可能原因有哪些？"
		case docTypeManual:
			q = topic + "的步骤是什么？"
		case docTypeRule:
			q = topic + "有哪些规则？"
		default:
			q = topic + "是什么意思？"
		}
		if _, ok := seen[q]; ok {
			continue
		}
		seen[q] = struct{}{}
		result = append(result, q)
		if len(result) >= 6 {
			break
		}
	}
	return result
}

// buildSummary 构造文档摘要：拼接主要章节标题 + 正文开头片段
func (p *ProfileLogicImpl) buildSummary(document *entity.Document, sectionTitles []string, parsedText string) string {
	var builder strings.Builder
	builder.WriteString("文档《")
	builder.WriteString(utils.BlankToDefault(strutil.Trim(document.DocumentName), "未命名文档"))
	builder.WriteString("》")
	if len(sectionTitles) > 0 {
		limit := max(len(sectionTitles), 4)
		builder.WriteString("主要涵盖：")
		builder.WriteString(strings.Join(sectionTitles[:limit], "、"))
		builder.WriteString("。")
	}
	excerpt := whitespaceRegexp.ReplaceAllString(strutil.Trim(parsedText), " ")
	if len(excerpt) > 180 {
		excerpt = excerpt[:180]
	}
	if strutil.IsNotBlank(excerpt) {
		builder.WriteString("摘要：")
		builder.WriteString(excerpt)
	}
	return strutil.Trim(builder.String())
}

// inferKnowledgeScopeCode 推断知识范围编码
func (p *ProfileLogicImpl) inferKnowledgeScopeCode(document *entity.Document, sectionTitles []string, parsedText string) string {
	combined := combinedText(document, parsedText, sectionTitles)
	switch {
	case containsAny(combined, "上线观察", "值班规则", "观察时长", "运营"):
		return scopeOperationRule
	case containsAny(combined, "机器人", "知识召回", "意图识别", "策略设计"):
		return scopeRobotStrategy
	case containsAny(combined, "安装", "部署", "默认密码", "访问地址"):
		return scopeDeployment
	case containsAny(combined, "故障", "排查", "异常", "检查顺序"):
		return scopeTroubleshoot
	case containsAny(combined, "产品简介", "核心特性", "技术规格", "产品概述"):
		return scopeProduct
	default:
		return scopeGeneral
	}
}

// inferKnowledgeScopeName 推断知识范围名称
func (p *ProfileLogicImpl) inferKnowledgeScopeName(scopeCode string) string {
	switch scopeCode {
	case scopeOperationRule:
		return "运营规则"
	case scopeRobotStrategy:
		return "机器人策略"
	case scopeDeployment:
		return "安装部署"
	case scopeTroubleshoot:
		return "故障排查"
	case scopeProduct:
		return "产品资料"
	default:
		return "通用文档"
	}
}

// inferBusinessCategory 推断业务分类
func (p *ProfileLogicImpl) inferBusinessCategory(documentType, parsedText string) string {
	switch documentType {
	case docTypeTroubleshoot:
		return "故障排查"
	case docTypeRule:
		return "规则"
	case docTypeSpec:
		return "规格说明"
	case docTypeManual:
		if containsAny(strings.ToLower(parsedText), "步骤", "操作", "部署") {
			return "操作手册"
		}
		return "手册"
	default:
		return "介绍"
	}
}

// buildDocumentTags 构建文档标签，逗号分隔
func (p *ProfileLogicImpl) buildDocumentTags(document *entity.Document, knowledgeScopeCode, documentType string, coreTopics []string) string {
	seen := make(map[string]struct{})
	result := make([]string, 0, 8)
	if strutil.IsNotBlank(document.DocumentTags) {
		for _, t := range strings.Split(document.DocumentTags, ",") {
			normalized := strutil.Trim(t)
			_, ok := seen[normalized]
			if normalized != "" && !ok {
				seen[normalized] = struct{}{}
				result = append(result, normalized)
			}
			if len(result) >= 8 {
				break
			}
		}
	}
	addTag := func(tag string) {
		normalized := strutil.Trim(tag)
		if strutil.IsBlank(normalized) {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		if len(result) >= 8 {
			return
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	addTag(knowledgeScopeCode)
	addTag(documentType)
	for i, topic := range coreTopics {
		if i >= 4 {
			break
		}
		addTag(topic)
	}
	return strings.Join(result, ",")
}

// ==================== 工具函数 ====================

func combinedText(document *entity.Document, parsedText string, sectionTitles []string) string {
	var builder strings.Builder
	builder.WriteString(strutil.Trim(document.DocumentName))
	builder.WriteString(" ")
	builder.WriteString(strutil.Trim(document.OriginalFileName))
	builder.WriteString(" ")
	builder.WriteString(strings.Join(sectionTitles, " "))
	builder.WriteString(" ")
	builder.WriteString(strutil.Trim(parsedText))
	return strings.ToLower(builder.String())
}

func containsAny(text string, values ...string) bool {
	normalized := strings.ToLower(strutil.Trim(text))
	for _, v := range values {
		if strings.Contains(normalized, strings.ToLower(strutil.Trim(v))) {
			return true
		}
	}
	return false
}

func stripSectionCode(title string) string {
	normalized := strutil.Trim(title)
	if normalized == "" {
		return normalized
	}
	return strutil.Trim(sectionCodePrefixRegexp.ReplaceAllString(normalized, ""))
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
