// 文档类型
/** 上传文档请求 */
export interface UploadDocumentReq {
  documentName?: string;
  operatorId?: string;
  knowledgeScopeCode?: string;
  knowledgeScopeName?: string;
  businessCategory?: string;
  documentTags?: string;
  file: File | null;
}

/** 分页查询文档列表请求 */
export interface QueryDocumentPageReq {
  pageNo?: number;
  pageSize?: number;
  keyword?: string;
}

/** 查询文档详情请求 */
export interface QueryDocumentDetailReq {
  documentId: string;
}

/** 删除文档请求 */
export interface DeleteDocumentReq {
  documentId: string;
}

/** 查询文档策略推荐结果请求 */
export interface QueryStrategyPlanReq {
  documentId: string;
}

/** 策略步骤子项（公共内嵌实体） */
export interface StrategyStepItem {
  stepNo: number;
  strategyType: 1 | 2 | 3;
}

/** 确认文档策略方案请求 */
export interface ConfirmStrategyReq {
  documentId: string;
  basePlanId: string;
  operatorId?: string;
  adjustNote?: string;
  parentSteps: StrategyStepItem[];
  childSteps: StrategyStepItem[];
}

/** 执行文档索引构建请求 */
export interface BuildIndexReq {
  documentId: string;
  planId: string;
  operatorId?: string;
}

/** 查询文档chunk列表请求 */
export interface QueryDocumentChunksReq {
  documentId: string;
  taskId?: string;
  pageNo?: number;
  pageSize?: number;
}

/** 查询单个文档chunk详情请求 */
export interface QueryDocumentChunkDetailReq {
  documentId: string;
  chunkId: string;
  taskId?: string;
}

/** 查询任务执行日志请求 */
export interface QueryTaskLogsReq {
  taskId: string;
  pageNo?: number;
  pageSize?: number;
}

/** 查询文档画像详情请求 */
export interface DocumentProfileDetailReq {
  documentId: string;
}

/** 重新生成文档画像请求 */
export interface DocumentProfileRegenerateReq {
  documentId: string;
  operatorId?: string;
}

/** 批量重新生成文档画像请求 */
export interface DocumentProfileBatchRegenerateReq {
  documentIds: string[];
  operatorId?: string;
}

// ====================== 内嵌基础响应子类型 ======================
/** 策略步骤 */
export interface DocumentStrategyStep {
  stepNo: number;
  pipelineType: string;
  pipelineTypeName: string;
  strategyType: number;
  strategyTypeName: string;
  strategyRole: number;
  strategyRoleName: string;
  sourceType: number;
  sourceTypeName: string;
  executeStatus: number;
  executeStatusName: string;
  recommendReason: string;
}

/** 策略流水线 */
export interface DocumentStrategyPipeline {
  pipelineType: string;
  pipelineTypeName: string;
  strategySnapshot: string;
  steps: DocumentStrategyStep[];
}

/** 策略方案 */
export interface DocumentStrategyPlan {
  planId: string;
  planVersion: number;
  planSource: number;
  planSourceName: string;
  planStatus: number;
  planStatusName: string;
  strategySnapshot: string;
  recommendReason: string;
  parentPipeline: DocumentStrategyPipeline | null;
  childPipeline: DocumentStrategyPipeline | null;
}

/** 文档父块 */
export interface DocumentParentBlockItem {
  parentBlockId: string;
  parentBlockNo: number;
  sectionPath: string;
  sourceType: number;
  sourceTypeName: string;
  charCount: number;
  tokenCount: number;
  childCount: number;
  startChunkNo: number;
  endChunkNo: number;
  parentText: string;
}

/** 文档分片 */
export interface DocumentChunkItem {
  chunkId: string;
  parentBlockId: string;
  parentBlockNo: number;
  parentChildCount: number;
  parentStartChunkNo: number;
  parentEndChunkNo: number;
  chunkNo: number;
  sectionPath: string;
  sourceType: number;
  sourceTypeName: string;
  charCount: number;
  tokenCount: number;
  vectorStatus: number;
  vectorStatusName: string;
  chunkText: string;
}

/** 任务日志 */
export interface TaskLog {
  id: string;
  stageType: number;
  stageTypeName: string;
  eventType: number;
  eventTypeName: string;
  logLevel: number;
  logLevelName: string;
  content: string;
  detailJson: string;
  createTime: string;
}

// ====================== 顶层响应类型 ======================
/** 上传文档响应 */
export interface UploadDocumentResp {
  documentId: string;
  taskId: string;
  documentName: string;
  parseStatus: number;
  strategyStatus: number;
  indexStatus: number;
}

/** 文档详情 */
export interface DocumentDetailResp {
  documentId: string;
  documentName: string;
  originalFileName: string;
  fileType: number;
  fileTypeName: string;
  fileSize: number;
  charCount: number;
  tokenCount: number;
  parseStatus: number;
  parseStatusName: string;
  strategyStatus: number;
  strategyStatusName: string;
  indexStatus: number;
  indexStatusName: string;
  parseErrorMsg: string;
  knowledgeScopeCode: string;
  knowledgeScopeName: string;
  businessCategory: string;
  documentTags: string;
  currentPlanId: string;
  lastIndexTaskId: string;
  latestTaskId: string;
  latestTaskType: number;
  latestTaskTypeName: string;
  latestTaskStatus: number;
  latestTaskStatusName: string;
  createTime: string;
  updateTime: string;
}

/** 分页查询文档列表响应 */
export interface QueryDocumentPageResp {
  pageNo: number;
  pageSize: number;
  total: number;
  records: DocumentDetailResp[];
}

/** 文档知识库选项响应 */
export interface KnowledgeDocumentOptionResp {
  documentId: string;
  documentName: string;
  knowledgeScopeName: string;
  businessCategory: string;
  documentTags: string[];
}

/** 删除文档响应 */
export interface DeleteDocumentResp {
  documentId: string;
  documentName: string;
}

/** 查询文档策略方案响应 */
export interface QueryStrategyPlanResp {
  documentId: string;
  documentName: string;
  parseStatus: number;
  parseStatusName: string;
  strategyStatus: number;
  strategyStatusName: string;
  indexStatus: number;
  indexStatusName: string;
  parseErrorMsg: string;
  planReady: boolean;
  plan: DocumentStrategyPlan | null;
}

/** 确认文档策略方案响应 */
export interface ConfirmStrategyResp {
  planId: string;
  documentId: string;
  planVersion: number;
  strategyStatus: number;
  strategyStatusName: string;
  normalized: boolean;
  parentPipeline: DocumentStrategyPipeline | null;
  childPipeline: DocumentStrategyPipeline | null;
}

/** 执行文档索引构建响应 */
export interface BuildIndexResp {
  taskId: string;
  documentId: string;
  taskType: number;
  taskTypeName: string;
  taskStatus: number;
  taskStatusName: string;
  indexStatus: number;
  indexStatusName: string;
}

/** 查询文档chunk列表响应 */
export interface QueryDocumentChunksResp {
  documentId: string;
  taskId: string;
  planId: string;
  pageNo: number;
  pageSize: number;
  total: number;
  records: DocumentChunkItem[];
}

/** 查询单个文档chunk详情响应 */
export interface QueryDocumentChunkDetailResp {
  documentId: string;
  taskId: string;
  planId: string;
  chunk: DocumentChunkItem | null;
  parentBlock: DocumentParentBlockItem | null;
  siblingChunks: DocumentChunkItem[];
}

/** 查询任务执行日志响应 */
export interface QueryTaskLogsResp {
  taskId: string;
  documentId: string;
  taskType: number;
  taskTypeName: string;
  taskStatus: number;
  taskStatusName: string;
  currentStage: number;
  currentStageName: string;
  startTime: string;
  finishTime: string;
  costMillis: number;
  errorCode: string;
  errorMsg: string;
  total: number;
  logs: TaskLog[];
}

/** 文档画像详情响应 */
export interface DocumentProfileResp {
  documentId: string;
  documentSummary: string;
  documentType: string;
  coreTopics: string;
  exampleQuestions: string;
  graphFriendly: number;
  supportsGraphOutline: number;
  supportsItemLookup: number;
  supportsGraphAssist: number;
  profileSource: string;
  profileStatus: number;
  errorMsg: string;
}

