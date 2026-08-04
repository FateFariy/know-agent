package logic

import (
	"context"
	"encoding/json"
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
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
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

var (
	sectionCodePrefixRegexp = regexp.MustCompile(`^(第[一二三四五六七八九十百0-9]+[章节条部分]\s*)|(\d+(?:\.\d+)+\s*)`)
	whitespaceRegexp        = regexp.MustCompile(`\s+`)
)

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
	profile := p.buildProfile(document, parsedText, structureNodes)

	existing, err := p.repo.SelectProfileByDocumentId(ctx, documentId)
	if err != nil && !errors.Is(err, errorx.ErrDocumentProfileNotFound) {
		return nil, err
	}

	if errors.Is(err, errorx.ErrDocumentProfileNotFound) {
		profile.ProfileVersion = 1
	} else {
		profile.ID = existing.ID
		profile.ProfileVersion = existing.ProfileVersion + 1
	}

	document = p.buildDocumentMetadata(document, profile, parsedText, structureNodes)
	fn := func(txCtx context.Context) error {
		if document != nil {
			if err = p.repo.UpdateDocumentById(txCtx, document); err != nil {
				return err
			}
		}
		return p.repo.SaveProfile(txCtx, profile)
	}
	if err = p.repo.Do(ctx, fn); err != nil {
		return nil, err
	}

	logx.Infof("文档画像生成完成: documentId=%d, documentType=%s, graphFriendly=%v, supportsItemLookup=%v, scopeCode=%s, businessCategory=%s, tags=%s",
		documentId, profile.DocumentType, profile.GraphFriendly, profile.SupportsItemLookup, document.KnowledgeScopeCode, document.BusinessCategory, document.DocumentTags)
	return profile, nil
}

// GetAllProfiles 根据文档ID获取画像
func (p *ProfileLogicImpl) GetAllProfiles(ctx context.Context) ([]*entity.DocumentProfile, error) {
	return p.repo.SelectDocumentProfiles(ctx)
}

// GetProfileByDocumentId 根据文档ID获取画像
func (p *ProfileLogicImpl) GetProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	return p.repo.SelectProfileByDocumentId(ctx, documentId)
}

// RegenerateProfile 重新生成文档画像
func (p *ProfileLogicImpl) RegenerateProfile(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	document, err := p.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}
	var parsedText string
	if strutil.IsNotBlank(document.ParseTextPath) {
		parsedText, err = p.port.DownloadText(ctx, document.ParseTextPath)
		if err != nil {
			return nil, err
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
		profile, err := p.RegenerateProfile(ctx, id)
		if err != nil {
			return result, err
		}
		result = append(result, profile)
	}
	return result, nil
}

// buildProfile 构建文档画像
func (p *ProfileLogicImpl) buildProfile(document *entity.Document, parsedText string, structureNodes []*entity.DocumentStructureNode) *entity.DocumentProfile {
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
	combined := combinedText(document, parsedText, sectionTitles)
	docType := vo.InferDocumentType(combined, supportsItemLookup)
	coreTopics := p.buildCoreTopics(document, sectionTitles)
	exampleQuestions := utils.DistinctFilterLimit(coreTopics, 6, func(t string) (string, bool) {
		return vo.ExampleQuestion(docType, t), true
	})
	profile := &entity.DocumentProfile{
		DocumentId:           document.ID,
		DocumentType:         docType,
		CoreTopics:           utils.ToCompactJSON(coreTopics),
		ExampleQuestions:     utils.ToCompactJSON(exampleQuestions),
		DocumentSummary:      p.buildSummary(document, sectionTitles, parsedText),
		SupportsGraphOutline: utils.Ternary(len(sectionTitles) >= 2, 1, 0),
		SupportsItemLookup:   utils.Ternary(supportsItemLookup, 1, 0),
		GraphFriendly:        utils.Ternary(len(sectionTitles) >= 2 && supportsItemLookup, 1, 0),
		SupportsGraphAssist:  1,
		ProfileSource:        profileSourceAuto,
		ProfileStatus:        profileStatusSuccess,
	}

	return profile
}

// buildDocumentMetadata 构建文档元数据
func (p *ProfileLogicImpl) buildDocumentMetadata(document *entity.Document, profile *entity.DocumentProfile, parsedText string, structureNodes []*entity.DocumentStructureNode) *entity.Document {
	sectionTitles := p.extractSectionTitles(structureNodes)
	combined := combinedText(document, parsedText, sectionTitles)
	code := klvo.KnowledgeScopeCode(combined)
	var coreTopics []string
	_ = json.Unmarshal([]byte(profile.CoreTopics), &coreTopics)

	businessCategory := vo.InferBusinessCategory(code, combined)
	scopeName := klvo.KnowledgeScopeName(code)
	updateDoc := &entity.Document{
		ID: document.ID,
	}

	changed := false
	if strutil.IsBlank(document.KnowledgeScopeCode) || strutil.IsBlank(document.KnowledgeScopeName) {
		updateDoc.KnowledgeScopeCode = code
		updateDoc.KnowledgeScopeName = scopeName
		changed = true
	}
	if strutil.IsBlank(document.BusinessCategory) {
		updateDoc.BusinessCategory = businessCategory
		changed = true
	}
	if strutil.IsBlank(document.DocumentTags) {
		updateDoc.FillDocumentTags(code, profile.DocumentType, coreTopics)
		changed = true
	}
	if changed {
		return updateDoc
	}
	return nil
}

// extractSectionTitles 提取章节标题（去重、取前 8 条）
func (p *ProfileLogicImpl) extractSectionTitles(structureNodes []*entity.DocumentStructureNode) []string {
	if len(structureNodes) == 0 {
		return nil
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

// buildCoreTopics 构建核心话题
func (p *ProfileLogicImpl) buildCoreTopics(document *entity.Document, sectionTitles []string) []string {
	fileName := strutil.Trim(document.DocumentName)
	fileTopic := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	sectionTitles = append(sectionTitles, fileTopic)
	return utils.DistinctFilterLimit(sectionTitles, 6, func(title string) (string, bool) {
		normalized := strutil.Trim(title)
		if normalized == "" {
			return normalized, false
		}
		return strutil.Trim(sectionCodePrefixRegexp.ReplaceAllString(normalized, "")), true
	})
}

// buildSummary 构造文档摘要：拼接主要章节标题 + 正文开头片段
func (p *ProfileLogicImpl) buildSummary(document *entity.Document, sectionTitles []string, parsedText string) string {
	var builder strings.Builder
	builder.WriteString("文档《")
	builder.WriteString(utils.BlankToDefault(strutil.Trim(document.DocumentName), "未命名文档"))
	builder.WriteString("》")
	if len(sectionTitles) > 0 {
		sectionTitles = utils.LimitSlice(sectionTitles, 4)
		builder.WriteString("主要涵盖：")
		builder.WriteString(strings.Join(sectionTitles, "、"))
		builder.WriteString("。")
	}
	excerpt := whitespaceRegexp.ReplaceAllString(strutil.Trim(parsedText), " ")
	if utils.Len(excerpt) > 180 {
		excerpt = strutil.Substring(excerpt, 0, 180)
	}
	if strutil.IsNotBlank(excerpt) {
		builder.WriteString("摘要：")
		builder.WriteString(excerpt)
	}
	return strutil.Trim(builder.String())
}

// ==================== 工具函数 ====================

func combinedText(document *entity.Document, parsedText string, sectionTitles []string) string {
	var builder strings.Builder
	builder.WriteString(strutil.Trim(document.DocumentName))
	builder.WriteString(strutil.Trim(document.OriginalFileName))
	builder.WriteString(strings.Join(sectionTitles, " "))
	builder.WriteString(strutil.Trim(parsedText))
	return builder.String()
}
