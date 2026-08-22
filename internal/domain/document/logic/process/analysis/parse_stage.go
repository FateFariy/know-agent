package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser/markdown"
)

// ParseStage 解析阶段：调用文本预处理逻辑解析文件内容
type ParseStage struct {
	repo     adapter.DocumentRepository
	registry *parse.Registry
}

func NewParseStage(repo adapter.DocumentRepository) *ParseStage {
	fallbackParser := &parser.TextParser{}
	parsers := []parse.Parser{
		&parser.HTMLParser{},
		&parser.TextParser{},
		&parser.PDFParser{},
		markdown.NewGoldmarkParser(),
	}

	return &ParseStage{
		repo:     repo,
		registry: parse.NewRegistry(fallbackParser, parsers...),
	}
}

func (p *ParseStage) Name() string {
	return "解析阶段"
}

func (p *ParseStage) Execute(ctx context.Context, parseCtx *Context) error {
	startTime := time.Now()

	analysisResult, err := p.process(ctx, parseCtx)
	if err != nil {
		return err
	}
	parseCtx.AnalysisResult = analysisResult
	structureCandidateCount := len(analysisResult.StructureNodes)
	// 记录"解析器返回结果"日志
	parserResultDetail, _ := json.Marshal(map[string]any{
		"artifactCount":           len(analysisResult.ParseArtifacts),
		"blockCount":              len(analysisResult.Blocks),
		"tableCandidateCount":     len(analysisResult.TableCandidates),
		"structureCandidateCount": structureCandidateCount,
		"parserCostMillis":        time.Since(startTime).Milliseconds(),
	})
	parserResultLog := &entity.DocumentTaskLog{
		TaskId:       parseCtx.TaskId,
		DocumentId:   parseCtx.DocumentId,
		StageType:    enum.TaskStageContentParse,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      fmt.Sprintf("解析器返回结果，结构候选 %d 个。", structureCandidateCount),
		DetailJson:   string(parserResultDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, parserResultLog)

	return nil
}

func (p *ParseStage) process(ctx context.Context, parseCtx *Context) (*aggregate.AnalysisResult, error) {
	if parseCtx == nil || parseCtx.Document == nil {
		return nil, fmt.Errorf("上下文或文档为空")
	}

	// 1. 块解析
	fileType := enum.FileTypeName(parseCtx.Document.FileType)
	contentParser := p.registry.Get(fileType)
	blocks, err := contentParser.Parse(ctx, parseCtx.RawFileBytes)
	if err != nil {
		return nil, err
	}
	blocks = blocks.Normalize()

	// 2. 提取解析文本
	parsedText := blocks.ExtractParsedText()

	// 3. 计算统计指标
	charCount := len([]rune(parsedText))
	headingCount, paragraphCount, maxParagraphLen := blocks.CalcStats()
	structureLevel := blocks.CalcStructureLevel()
	contentQualityLevel := blocks.CalcContentQualityLevel()

	// 4. 结构节点投影
	structureNodes := projectStructureNodes(parseCtx.Document.OriginalFileName, blocks)

	// 5. 表格候选投影
	tableCandidates := projectTableCandidates(blocks)

	// 6. 解析产物生成
	parseArtifacts := buildParseArtifacts(parseCtx.Document.OriginalFileName, blocks, structureNodes)

	return &aggregate.AnalysisResult{
		ParsedText:          parsedText,
		CharCount:           charCount,
		TokenCount:          utils.EstimateTokens(parsedText),
		StructureLevel:      structureLevel,
		ContentQualityLevel: contentQualityLevel,
		HeadingCount:        headingCount,
		ParagraphCount:      paragraphCount,
		MaxParagraphLength:  maxParagraphLen,
		StructureNodes:      structureNodes,
		TableCandidates:     tableCandidates,
		ParseArtifacts:      parseArtifacts,
		Blocks:              blocks,
	}, nil
}

// calcContentQualityLevel 计算内容质量等级
func calcContentQualityLevel(charCount, blockCount, maxParagraphLen int) int {
	if charCount >= 5000 && blockCount >= 10 && maxParagraphLen >= 100 {
		return enum.ContentQualityLevelHigh
	}
	if charCount >= 1000 && blockCount >= 3 {
		return enum.ContentQualityLevelMedium
	}
	if charCount >= 100 {
		return enum.ContentQualityLevelLow
	}
	return 0
}

// projectStructureNodes 结构节点投影：基于块列表构建文档结构树
func projectStructureNodes(documentTitle string, blocks entity.DocumentBlocks) []*entity.StructureNode {
	if len(blocks) == 0 {
		return nil
	}

	var nodes entity.StructureNodes
	var sectionStack []int // 存储 Section 节点的 NodeNo

	// 创建根节点（文档）
	rootNode := &entity.StructureNode{
		NodeNo:        1,
		NodeType:      vo.NodeTypeDocument,
		Depth:         0,
		Title:         documentTitle,
		AnchorText:    utils.ClipHead(documentTitle, 200),
		CanonicalPath: "/",
		SectionPath:   "",
		ContentText:   documentTitle,
	}
	nodes = append(nodes, rootNode)
	currentNodeNo := 1
	sectionStack = append(sectionStack, 1)

	for _, block := range blocks {
		if block == nil {
			continue
		}

		blockText := strings.TrimSpace(block.Text)
		if blockText == "" && block.TableHTML == "" && block.ImageCaption == "" {
			continue
		}

		switch block.BlockType {
		case enum.BlockTypeTitle:
			// 计算标题层级
			level := block.HeadingLevel()
			// 弹出层级 >= 当前标题的栈顶元素
			for len(sectionStack) > 1 {
				topIdx := len(sectionStack) - 1
				topNode := nodes.FindNodeByNo(sectionStack[topIdx])
				if topNode != nil && topNode.Depth >= level {
					sectionStack = sectionStack[:topIdx]
				} else {
					break
				}
			}

			parentNodeNo := sectionStack[len(sectionStack)-1]
			parentNode := nodes.FindNodeByNo(parentNodeNo)

			currentNodeNo++
			sectionNode := &entity.StructureNode{
				NodeNo:              currentNodeNo,
				NodeType:            vo.NodeTypeSection,
				ParentNodeNo:        parentNodeNo,
				Depth:               parentNode.Depth + 1,
				NodeCode:            block.ExtractHeadingCode(),
				Title:               blockText,
				AnchorText:          utils.ClipHead(blockText, 200),
				CanonicalPath:       block.CanonicalPath,
				SectionPath:         block.SectionPath,
				ContentText:         blockText,
				SyntaxSchemaVersion: utils.GetStringFromMap(block.Metadata, "syntaxSchemaVersion", ""),
				SyntaxNodeId:        utils.GetStringFromMap(block.Metadata, "syntaxNodeId", ""),
				SyntaxNodeType:      utils.GetStringFromMap(block.Metadata, "syntaxNodeType", ""),
				SyntaxSourceOrigin:  utils.GetStringFromMap(block.Metadata, "sourceOrigin", ""),
			}
			// 设置兄弟关系
			siblings := nodes.FindChildrenByNo(parentNodeNo)
			if len(siblings) > 0 {
				sectionNode.PrevSiblingNodeNo = siblings[len(siblings)-1]
				prevSibling := nodes.FindNodeByNo(siblings[len(siblings)-1])
				if prevSibling != nil {
					prevSibling.NextSiblingNodeNo = currentNodeNo
				}
			}
			nodes = append(nodes, sectionNode)
			sectionStack = append(sectionStack, currentNodeNo)

		default:
			// 内容块归入当前最近的 Section
			currentSectionNo := sectionStack[len(sectionStack)-1]
			contentParentNo := currentSectionNo
			contentParent := nodes.FindNodeByNo(contentParentNo)

			currentNodeNo++
			contentNode := &entity.StructureNode{
				NodeNo:        currentNodeNo,
				NodeType:      vo.NodeTypeStep,
				ParentNodeNo:  contentParentNo,
				Depth:         contentParent.Depth + 1,
				Title:         utils.FirstNonBlank(blockText, block.ImageCaption, block.ExtractTableSummary()),
				AnchorText:    utils.ClipHead(blockText, 200),
				CanonicalPath: block.CanonicalPath,
				SectionPath:   block.SectionPath,
				ContentText:   block.BuildContentText(),
			}
			// 设置兄弟关系
			siblings := nodes.FindChildrenByNo(contentParentNo)
			if len(siblings) > 0 {
				contentNode.PrevSiblingNodeNo = siblings[len(siblings)-1]
				prevSibling := nodes.FindNodeByNo(siblings[len(siblings)-1])
				if prevSibling != nil {
					prevSibling.NextSiblingNodeNo = currentNodeNo
				}
			}
			nodes = append(nodes, contentNode)
		}
	}

	return nodes
}

// projectTableCandidates 表格候选投影
func projectTableCandidates(blocks entity.DocumentBlocks) []*shared.TableCandidate {
	var candidates []*shared.TableCandidate
	for _, block := range blocks {
		if block == nil || block.BlockType != enum.BlockTypeTable {
			continue
		}
		if block.TableHTML == "" && len(block.TableRows) == 0 {
			continue
		}

		tableCandidate := &shared.TableCandidate{
			SourceBlockNo:   block.BlockNo,
			SectionPath:     block.SectionPath,
			PageNo:          block.PageNo,
			PageRange:       block.PageRange,
			BoundingBoxJson: block.BboxJson,
			TableHTML:       block.TableHTML,
			TitleHint:       block.ImageCaption,
			ProjectionOwner: "GO_BLOCKS_PROJECTION",
			Rows:            buildTableRows(block.TableRows),
			SourceMetadata:  block.Metadata,
		}
		candidates = append(candidates, tableCandidate)
	}
	return candidates
}

// buildTableRows 构建表格行
func buildTableRows(rawRows [][]string) []*shared.TableRow {
	if len(rawRows) == 0 {
		return nil
	}
	var rows []*shared.TableRow
	for i, rawRow := range rawRows {
		cells := buildTableCells(rawRow)
		isHeader := i == 0
		for _, cell := range cells {
			if cell.IsHeader {
				isHeader = true
				break
			}
		}
		row := &shared.TableRow{
			RowIndex: i,
			IsHeader: isHeader,
			Cells:    cells,
		}
		rows = append(rows, row)
	}
	return rows
}

// buildTableCells 构建表格单元格
func buildTableCells(rawRow []string) []*shared.TableCell {
	var cells []*shared.TableCell
	for j, text := range rawRow {
		cells = append(cells, &shared.TableCell{
			RowIndex:    0,
			ColumnIndex: j,
			Text:        text,
		})
	}
	return cells
}

// buildParseArtifacts 构建解析产物
func buildParseArtifacts(fileName string, blocks entity.DocumentBlocks, structureNodes []*entity.StructureNode) []*entity.ParseArtifact {
	var artifacts []*entity.ParseArtifact

	// 构建哈希输入
	hashParts := []string{fileName}
	for _, block := range blocks {
		if block != nil {
			hashParts = append(hashParts, block.Text)
		}
	}
	for _, node := range structureNodes {
		if node != nil {
			hashParts = append(hashParts, node.Title)
		}
	}
	contentHash := utils.CalcContentHash(hashParts...)

	// 基础解析信息产物
	parserArtifact := &entity.ParseArtifact{
		ArtifactType:  "DOCUMENT_PARSE_INFO",
		ObjectName:    fileName,
		ContentHash:   contentHash,
		ParserName:    "go-native-parser",
		ParserVersion: "1.0.0",
		FileName:      fileName + ".parse-info.json",
		ContentType:   "application/json;charset=UTF-8",
	}
	artifacts = append(artifacts, parserArtifact)

	// 如果有结构节点，添加结构快照
	if len(structureNodes) > 0 {
		structureArtifact := &entity.ParseArtifact{
			ArtifactType:  "STRUCTURE_SNAPSHOT",
			ObjectName:    fileName,
			ContentHash:   contentHash,
			ParserName:    "go-native-parser",
			ParserVersion: "1.0.0",
			FileName:      fileName + ".structure.json",
			ContentType:   "application/json;charset=UTF-8",
		}
		artifacts = append(artifacts, structureArtifact)
	}

	return artifacts
}
