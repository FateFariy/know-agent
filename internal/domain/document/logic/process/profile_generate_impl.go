package process

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// 画像状态常量
const (
	profileStatusSuccess = 2
	profileSourceAuto    = "auto"
)

var (
	sectionCodePrefixRegexp = regexp.MustCompile(`^(第[一二三四五六七八九十百0-9]+[章节条部分]\s*)|(\d+(?:\.\d+)+\s*)`)
	whitespaceRegexp        = regexp.MustCompile(`\s+`)
)

type ProfileGenerateImpl struct {
	repo adapter.DocumentRepository
}

func NewProfileGenerateImpl(repo adapter.DocumentRepository) *ProfileGenerateImpl {
	return &ProfileGenerateImpl{
		repo: repo,
	}
}

func (p *ProfileGenerateImpl) Generate(ctx context.Context, documentId int64, analysisResult *aggregate.AnalysisResult, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error) {
	if documentId == 0 {
		return nil, errors.New("documentId 不能为空")
	}
	document, err := p.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}

	parsedText := ""
	if analysisResult != nil {
		parsedText = utils.Trim(analysisResult.ParsedText)
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

	if err = p.repo.SaveProfile(ctx, profile); err != nil {
		return nil, err
	}

	logx.Infof("文档画像生成完成: documentId=%d, documentType=%s, graphFriendly=%v, supportsItemLookup=%v",
		documentId, profile.DocumentType, profile.GraphFriendly, profile.SupportsItemLookup)
	return profile, nil
}

// buildProfile 构建文档画像
func (p *ProfileGenerateImpl) buildProfile(document *entity.Document, parsedText string, structureNodes entity.StructureNodes) *entity.DocumentProfile {
	sectionTitles := structureNodes.ExtractSectionTitles()
	supportsItemLookup := false
	for _, node := range structureNodes {
		if node == nil {
			continue
		}
		if node.NodeType == enum.NodeTypeStep || node.NodeType == enum.NodeTypeListItem {
			supportsItemLookup = true
			break
		}
	}
	docType := resolveStructuralDocumentType(sectionTitles, supportsItemLookup, len(sectionTitles) >= 2)
	coreTopics := p.buildCoreTopics(document, sectionTitles)
	exampleQuestions := utils.FilterMapUniqueLimit(coreTopics, 6, func(topic string) (string, string, bool) {
		return topic, topic + "包含哪些内容？", true
	})
	profile := &entity.DocumentProfile{
		DocumentId:           document.ID,
		DocumentType:         docType,
		CoreTopics:           utils.ToCompactJSON(coreTopics),
		ExampleQuestions:     utils.ToCompactJSON(exampleQuestions),
		DocumentSummary:      p.buildSummary(sectionTitles, parsedText),
		SupportsGraphOutline: utils.Ternary(len(sectionTitles) >= 2, 1, 0),
		SupportsItemLookup:   utils.Ternary(supportsItemLookup, 1, 0),
		GraphFriendly:        utils.Ternary(supportsItemLookup || len(sectionTitles) >= 2, 1, 0),
		SupportsGraphAssist:  1,
		ProfileSource:        profileSourceAuto,
		ProfileStatus:        profileStatusSuccess,
	}
	return profile
}

// buildCoreTopics 构建核心话题
func (p *ProfileGenerateImpl) buildCoreTopics(document *entity.Document, sectionTitles []string) []string {
	fileName := utils.Trim(document.DocumentName)
	fileTopic := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	sectionTitles = append(sectionTitles, fileTopic)
	return utils.FilterUniqueLimit(sectionTitles, 6, func(title string) (string, bool) {
		normalized := utils.Trim(title)
		if normalized == "" {
			return normalized, false
		}
		return utils.Trim(sectionCodePrefixRegexp.ReplaceAllString(normalized, "")), true
	})
}

// buildSummary 构造文档摘要：拼接主要章节标题 + 正文开头片段
func (p *ProfileGenerateImpl) buildSummary(sectionTitles []string, parsedText string) string {
	var builder strings.Builder
	if len(sectionTitles) > 0 {
		sectionTitles = utils.Limit(sectionTitles, 4)
		builder.WriteString("主要涵盖：")
		builder.WriteString(strings.Join(sectionTitles, "、"))
		builder.WriteString("。")
	}
	excerpt := whitespaceRegexp.ReplaceAllString(utils.Trim(parsedText), " ")
	if utils.Len(excerpt) > 180 {
		excerpt = utils.Substring(excerpt, 0, 180)
	}
	if utils.IsNotBlank(excerpt) {
		builder.WriteString("摘要：")
		builder.WriteString(excerpt)
	}
	return utils.Trim(builder.String())
}

// resolveStructuralDocumentType 根据文档的结构特征判定其结构化类型
func resolveStructuralDocumentType(sectionTitles []string, supportsItemLookup bool, supportsGraphOutline bool) string {
	switch {
	case supportsItemLookup: //支持条目级检索
		return "structured_items"
	case supportsGraphOutline: //支持图结构大纲
		return "structured_outline"
	case len(sectionTitles) > 0: //存在章节标题
		return "structured_section"
	default:
		return "plain_text"
	}
}
