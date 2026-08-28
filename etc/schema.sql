-- 1. 通道执行记录
CREATE TABLE IF NOT EXISTS `chat_channel_execution`
(
    `id`                   BIGINT         NOT NULL COMMENT '主键ID',
    `conversation_id`      VARCHAR(64)    NOT NULL COMMENT '对话ID',
    `exchange_id`          BIGINT         NOT NULL COMMENT '交换ID',
    `trace_id`             VARCHAR(128)   NOT NULL COMMENT '跟踪ID',
    `sub_question_index`   INT            NOT NULL DEFAULT 0 COMMENT '子问题索引',
    `sub_question`         TEXT COMMENT '子问题',
    `channel_type`         VARCHAR(32)    NOT NULL COMMENT '渠道类型',
    `execution_state`      INT            NOT NULL DEFAULT 0 COMMENT '执行状态',
    `start_time`           DATETIME       NOT NULL COMMENT '开始时间',
    `end_time`             DATETIME COMMENT '结束时间',
    `duration_ms`          BIGINT         NOT NULL DEFAULT 0 COMMENT '执行时长（毫秒）',
    `recalled_count`       INT            NOT NULL DEFAULT 0 COMMENT '召回数量',
    `accepted_count`       INT            NOT NULL DEFAULT 0 COMMENT '接受数量',
    `final_selected_count` INT            NOT NULL DEFAULT 0 COMMENT '最终选中数量',
    `avg_score`            DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '平均分数',
    `max_score`            DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '最大分数',
    `min_score`            DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '最小分数',
    `config_snapshot`      TEXT COMMENT '配置快照',
    `error_message`        TEXT COMMENT '错误信息',
    `create_time`          DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`          DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`              TINYINT        NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`),
    INDEX `idx_exchange_id` (`exchange_id`),
    INDEX `idx_trace_id` (`trace_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='通道执行记录';

-- 2. 会话记录表
CREATE TABLE IF NOT EXISTS `chat_dialogue`
(
    `id`                                 BIGINT       NOT NULL COMMENT '主键ID',
    `conversation_id`                    VARCHAR(64)  NOT NULL COMMENT '会话ID',
    `session_status`                     INT          NOT NULL DEFAULT 0 COMMENT '会话状态',
    `chat_mode`                          INT          NOT NULL DEFAULT 0 COMMENT '聊天模式',
    `selected_document_id`               BIGINT       NOT NULL DEFAULT 0 COMMENT '选中的文档ID',
    `selected_document_name`             VARCHAR(255) NOT NULL DEFAULT '' COMMENT '选中的文档名称',
    `knowledge_base_selection_mode`      VARCHAR(50)  NOT NULL DEFAULT '' COMMENT '知识库选择模式',
    `selected_knowledge_base_ids_json`   JSON COMMENT '选中知识库ID列表JSON',
    `selected_knowledge_base_names_json` JSON COMMENT '选中知识库名称列表JSON',
    `create_time`                        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`                        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                            TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='会话记录表';

-- 3. 对话记录表（交换）
CREATE TABLE IF NOT EXISTS `chat_exchange`
(
    `id`                                 BIGINT      NOT NULL COMMENT '主键ID',
    `conversation_id`                    VARCHAR(64) NOT NULL COMMENT '会话ID',
    `question`                        TEXT        NOT NULL COMMENT '用户提问',
    `answer`                      TEXT        NOT NULL COMMENT '回复内容',
    `turn_status`                     TINYINT     NOT NULL DEFAULT 0 COMMENT '交互状态',
    `thinking_steps`                     JSON COMMENT '思维步骤',
    `references`                         JSON COMMENT '参考列表',
    `recommendations`                    JSON COMMENT '推荐问题列表',
    `used_tools`                         JSON COMMENT '工具使用列表',
    `debug_trace_json`                   TEXT COMMENT '调试跟踪JSON',
    `error_message`                        VARCHAR(500) COMMENT '错误信息',
    `first_response_time_ms`             BIGINT      NOT NULL DEFAULT 0 COMMENT '首包响应耗时(ms)',
    `total_latency_ms`                   BIGINT      NOT NULL DEFAULT 0 COMMENT '总响应耗时(ms)',
    `knowledge_base_selection_mode`      VARCHAR(50) NOT NULL DEFAULT '' COMMENT '知识库选择模式',
    `selected_knowledge_base_ids_json`   JSON COMMENT '选中知识库ID列表JSON',
    `selected_knowledge_base_names_json` JSON COMMENT '选中知识库名称列表JSON',
    `retrieval_config_snapshot_json`     JSON COMMENT '检索配置快照JSON',
    `create_time`                        DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`                        DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                            TINYINT     NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='对话记录表';

-- 4. 追踪阶段表
CREATE TABLE IF NOT EXISTS `chat_exchange_trace_stage`
(
    `id`              BIGINT       NOT NULL COMMENT '主键ID',
    `conversation_id` VARCHAR(64)  NOT NULL COMMENT '对话ID',
    `exchange_id`     BIGINT       NOT NULL COMMENT '交互ID',
    `trace_id`        VARCHAR(64)  NOT NULL COMMENT '追踪ID',
    `stage_code`      VARCHAR(50)  NOT NULL COMMENT '阶段编码',
    `stage_name`      VARCHAR(100) NOT NULL COMMENT '阶段名称',
    `stage_order`     INT          NOT NULL DEFAULT 0 COMMENT '阶段顺序',
    `stage_level`     INT          NOT NULL DEFAULT 0 COMMENT '阶段层级',
    `parent_stage_id` BIGINT       NOT NULL DEFAULT 0 COMMENT '父阶段ID',
    `execution_mode`  VARCHAR(50)  NOT NULL DEFAULT '' COMMENT '执行模式',
    `stage_state`     TINYINT      NOT NULL DEFAULT 0 COMMENT '阶段状态',
    `start_time`      DATETIME COMMENT '开始时间',
    `end_time`        DATETIME COMMENT '结束时间',
    `duration_ms`     BIGINT       NOT NULL DEFAULT 0 COMMENT '耗时(ms)',
    `summary_text`    TEXT COMMENT '阶段摘要',
    `error_message`   VARCHAR(500) COMMENT '错误信息',
    `snapshot_json`   TEXT COMMENT '快照JSON',
    `create_time`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`         TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`),
    INDEX `idx_exchange_id` (`exchange_id`),
    INDEX `idx_trace_id` (`trace_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='追踪阶段表';

-- 5. 会话记忆摘要
CREATE TABLE IF NOT EXISTS `chat_memory_summary`
(
    `id`                      BIGINT      NOT NULL COMMENT '主键ID',
    `conversation_id`         VARCHAR(64) NOT NULL COMMENT '对话ID',
    `covered_exchange_id`     BIGINT      NOT NULL COMMENT '覆盖的交互ID',
    `covered_exchange_count`  INT         NOT NULL DEFAULT 0 COMMENT '覆盖交互数量',
    `compression_count`       INT         NOT NULL DEFAULT 0 COMMENT '压缩次数',
    `summary_version`         INT         NOT NULL DEFAULT 0 COMMENT '摘要版本',
    `summary_text`            TEXT        NOT NULL COMMENT '摘要文本',
    `summary_json`            TEXT COMMENT '摘要JSON',
    `last_source_update_time` DATETIME    NOT NULL COMMENT '源数据最后编辑时间',
    `create_time`             DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`             DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                 TINYINT     NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='会话记忆摘要';

-- 6. 检索结果表
CREATE TABLE IF NOT EXISTS `chat_retrieval_result`
(
    `id`                       BIGINT         NOT NULL COMMENT '主键ID',
    `conversation_id`          VARCHAR(64)    NOT NULL COMMENT '对话ID',
    `exchange_id`              BIGINT         NOT NULL COMMENT '交互ID',
    `trace_id`                 VARCHAR(64)    NOT NULL COMMENT '追踪ID',
    `sub_question_index`       INT            NOT NULL DEFAULT 0 COMMENT '子问题索引',
    `sub_question`             TEXT COMMENT '子问题',
    `candidate_id`             VARCHAR(64)    NOT NULL COMMENT '候选ID',
    `channel_type`             VARCHAR(50)    NOT NULL COMMENT '通道类型',
    `channel_rank`             INT            NOT NULL DEFAULT 0 COMMENT '通道排名',
    `rrf_rank`                 INT            NOT NULL DEFAULT 0 COMMENT 'RRF排名',
    `final_rank`               INT            NOT NULL DEFAULT 0 COMMENT '最终排名',
    `original_score`           DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '原始分数',
    `rrf_score`                DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT 'RRF分数',
    `hybrid_score`             DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '混合分数',
    `metadata_boost`           DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '元数据增强分数',
    `vector_score`             DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '向量分数',
    `keyword_score`            DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '关键词分数',
    `rerank_score`             DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '重排分数',
    `gate_passed`              TINYINT        NOT NULL DEFAULT 0 COMMENT '是否通过门控',
    `is_elevated`              TINYINT        NOT NULL DEFAULT 0 COMMENT '是否提升',
    `is_selected`              TINYINT        NOT NULL DEFAULT 0 COMMENT '是否被选中',
    `selection_reason`         VARCHAR(500) COMMENT '选中原因',
    `filtered_reason`          VARCHAR(500) COMMENT '过滤原因',
    `rank_feature`             TEXT COMMENT '排序特征',
    `document_id`              BIGINT         NOT NULL COMMENT '文档ID',
    `document_name`            VARCHAR(255)   NOT NULL COMMENT '文档名称',
    `chunk_id`                 BIGINT         NOT NULL COMMENT '文本块ID',
    `chunk_type`               VARCHAR(50)    NOT NULL COMMENT '文本块类型',
    `chunk_no`                 INT            NOT NULL DEFAULT 0 COMMENT '文本块序号',
    `parent_block_id`          BIGINT         NOT NULL DEFAULT 0 COMMENT '父块ID',
    `parent_block_no`          INT            NOT NULL DEFAULT 0 COMMENT '父块序号',
    `section_path`             VARCHAR(500) COMMENT '章节路径',
    `chunk_text_preview`       TEXT COMMENT '文本块预览',
    `chunk_char_count`         INT            NOT NULL DEFAULT 0 COMMENT '文本块字符数',
    `context_identity`         VARCHAR(255) COMMENT '上下文标识',
    `citation_identity`        VARCHAR(255) COMMENT '引用标识',
    `citation_identity_hash`   VARCHAR(64) COMMENT '引用标识哈希',
    `citation_evidence_type`   VARCHAR(50) COMMENT '引用证据类型',
    `context_only`             TINYINT        NOT NULL DEFAULT 0 COMMENT '仅上下文',
    `source_evidence_resolved` TINYINT        NOT NULL DEFAULT 0 COMMENT '源证据已解析',
    `create_time`              DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`              DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                  TINYINT        NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`),
    INDEX `idx_exchange_id` (`exchange_id`),
    INDEX `idx_trace_id` (`trace_id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_chunk_id` (`chunk_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='检索结果表';

-- 6.5 RAG 质量评估表
CREATE TABLE IF NOT EXISTS `chat_exchange_eval`
(
    `id`               BIGINT       NOT NULL COMMENT '主键ID',
    `conversation_id`  VARCHAR(64)  NOT NULL COMMENT '对话ID',
    `exchange_id`      BIGINT       NOT NULL COMMENT '交互ID',
    `metric_name`      VARCHAR(64)  NOT NULL COMMENT '指标编码(answer_faithfulness/answer_relevancy/context_precision)',
    `metric_label`     VARCHAR(64)  NOT NULL COMMENT '指标展示名',
    `score`            DECIMAL(10,4) NOT NULL DEFAULT 0 COMMENT '评估得分(0~1)',
    `latency_ms`       BIGINT       NOT NULL DEFAULT 0 COMMENT '评估耗时(ms)',
    `status`           TINYINT      NOT NULL DEFAULT 0 COMMENT '评估状态(0:成功,1:失败)',
    `error_msg`        VARCHAR(512) COMMENT '错误信息',
    `create_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`          TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_exchange` (`conversation_id`, `exchange_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='RAG质量评估表';

-- 7. 文档主表
CREATE TABLE IF NOT EXISTS `document`
(
    `id`                    BIGINT       NOT NULL COMMENT '主键ID',
    `document_name`         VARCHAR(255) NOT NULL COMMENT '文档名称',
    `original_file_name`    VARCHAR(255) NOT NULL COMMENT '原始文件名',
    `file_type`             INT          NOT NULL DEFAULT 0 COMMENT '文件类型',
    `mime_type`             VARCHAR(100) NOT NULL COMMENT 'MIME类型',
    `file_size`             BIGINT       NOT NULL DEFAULT 0 COMMENT '文件大小',
    `storage_type`          INT          NOT NULL DEFAULT 0 COMMENT '存储类型',
    `bucket_name`           VARCHAR(100) NOT NULL COMMENT '存储桶名称',
    `object_name`           VARCHAR(255) NOT NULL COMMENT '对象名称',
    `object_url`            VARCHAR(500) NOT NULL COMMENT '对象URL',
    `parse_status`          INT          NOT NULL DEFAULT 0 COMMENT '解析状态',
    `strategy_status`       INT          NOT NULL DEFAULT 0 COMMENT '策略状态',
    `index_status`          INT          NOT NULL DEFAULT 0 COMMENT '索引状态',
    `char_count`            INT          NOT NULL DEFAULT 0 COMMENT '字符数',
    `token_count`           INT          NOT NULL DEFAULT 0 COMMENT 'Token数',
    `structure_level`       INT          NOT NULL DEFAULT 0 COMMENT '结构层级',
    `content_quality_level` INT          NOT NULL DEFAULT 0 COMMENT '内容质量等级',
    `parse_text_path`       VARCHAR(500) NOT NULL COMMENT '解析文本路径',
    `parse_error_msg`       TEXT COMMENT '解析错误信息',
    `knowledge_base_id`     BIGINT       NOT NULL COMMENT '知识库ID',
    `knowledge_base_name`   VARCHAR(255) NOT NULL COMMENT '知识库名称',
    `current_plan_id`       BIGINT       NOT NULL DEFAULT 0 COMMENT '当前计划ID',
    `last_parse_task_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '最后解析任务ID',
    `structure_node_count`  INT          NOT NULL DEFAULT 0 COMMENT '结构节点数',
    `last_index_task_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '最后索引任务ID',
    `create_time`           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`               TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_knowledge_base_id` (`knowledge_base_id`),
    INDEX `idx_document_name` (`document_name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档主表';

-- 8. 文档块表
CREATE TABLE IF NOT EXISTS `document_block`
(
    `id`                  BIGINT      NOT NULL COMMENT '主键ID',
    `document_id`         BIGINT      NOT NULL COMMENT '文档ID',
    `task_id`             BIGINT      NOT NULL COMMENT '任务ID',
    `block_no`            INT         NOT NULL DEFAULT 0 COMMENT '块序号',
    `block_type`          VARCHAR(50) NOT NULL COMMENT '块类型',
    `parent_block_id`     BIGINT      NOT NULL DEFAULT 0 COMMENT '父块ID',
    `section_path`        VARCHAR(500) COMMENT '章节路径',
    `canonical_path`      VARCHAR(500) COMMENT '规范路径',
    `page_no`             INT         NOT NULL DEFAULT 0 COMMENT '页码',
    `page_range`          VARCHAR(50) COMMENT '页码范围',
    `bbox_json`           TEXT COMMENT '边界框JSON',
    `text`                TEXT        NOT NULL COMMENT '文本内容',
    `content_with_weight` TEXT COMMENT '加权内容',
    `table_html`          TEXT COMMENT '表格HTML',
    `image_object_name`   VARCHAR(255) COMMENT '图片对象名',
    `image_caption`       TEXT COMMENT '图片标题',
    `metadata_json`       TEXT COMMENT '元数据JSON',
    `create_time`         DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`         DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`             TINYINT     NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档块表';

-- 9. 文档文本块表
CREATE TABLE IF NOT EXISTS `document_chunk`
(
    `id`                  BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`         BIGINT   NOT NULL COMMENT '文档ID',
    `task_id`             BIGINT   NOT NULL COMMENT '任务ID',
    `plan_id`             BIGINT   NOT NULL COMMENT '计划ID',
    `parent_chunk_id`     BIGINT   NOT NULL DEFAULT 0 COMMENT '父块ID',
    `chunk_no`            INT      NOT NULL DEFAULT 0 COMMENT '块序号',
    `source_type`         INT      NOT NULL DEFAULT 0 COMMENT '来源类型',
    `section_path`        TEXT COMMENT '章节路径',
    `structure_node_id`   BIGINT   NOT NULL DEFAULT 0 COMMENT '结构节点ID',
    `structure_node_type` INT      NOT NULL DEFAULT 0 COMMENT '结构节点类型',
    `canonical_path`      TEXT COMMENT '规范路径',
    `item_index`          INT      NOT NULL DEFAULT 0 COMMENT '项索引',
    `chunk_text`          TEXT     NOT NULL COMMENT '块文本',
    `char_count`          INT      NOT NULL DEFAULT 0 COMMENT '字符数',
    `token_count`         INT      NOT NULL DEFAULT 0 COMMENT 'Token数',
    `vector_status`       INT      NOT NULL DEFAULT 0 COMMENT '向量状态',
    `vector_store_type`   INT      NOT NULL DEFAULT 0 COMMENT '向量存储类型',
    `vector_id`           VARCHAR(255) COMMENT '向量ID',
    `content_with_weight` TEXT COMMENT '加权内容',
    `chunk_type`          VARCHAR(50) COMMENT '块类型',
    `title`               VARCHAR(500) COMMENT '标题',
    `keywords`            TEXT COMMENT '关键词',
    `questions`           TEXT COMMENT '预设问题',
    `page_no`             INT      NOT NULL DEFAULT 0 COMMENT '页码',
    `page_range`          VARCHAR(50) COMMENT '页码范围',
    `bbox_json`           TEXT COMMENT '边界框JSON',
    `source_block_ids`    TEXT COMMENT '源块ID列表',
    `create_time`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`             TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_plan_id` (`plan_id`),
    INDEX `idx_parent_chunk_id` (`parent_chunk_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档文本块表';

-- 10. 文档父块表
CREATE TABLE IF NOT EXISTS `document_parent_chunk`
(
    `id`                  BIGINT       NOT NULL COMMENT '主键ID',
    `document_id`         BIGINT       NOT NULL COMMENT '文档ID',
    `task_id`             BIGINT       NOT NULL COMMENT '任务ID',
    `plan_id`             BIGINT       NOT NULL COMMENT '计划ID',
    `parent_no`           INT          NOT NULL DEFAULT 0 COMMENT '父节点编号',
    `source_type`         INT          NOT NULL DEFAULT 0 COMMENT '来源类型',
    `section_path`        VARCHAR(255) COMMENT '章节路径',
    `structure_node_id`   BIGINT       NOT NULL DEFAULT 0 COMMENT '结构节点ID',
    `structure_node_type` INT          NOT NULL DEFAULT 0 COMMENT '结构节点类型',
    `canonical_path`      VARCHAR(255) COMMENT '规范路径',
    `item_index`          INT          NOT NULL DEFAULT 0 COMMENT '项目索引',
    `parent_text`         VARCHAR(255) NOT NULL COMMENT '父节点文本',
    `char_count`          INT          NOT NULL DEFAULT 0 COMMENT '字符数',
    `token_count`         INT          NOT NULL DEFAULT 0 COMMENT '令牌数',
    `child_count`         INT          NOT NULL DEFAULT 0 COMMENT '子节点数',
    `start_chunk_no`      INT          NOT NULL DEFAULT 0 COMMENT '起始块号',
    `end_chunk_no`        INT          NOT NULL DEFAULT 0 COMMENT '结束块号',
    `page_range`          VARCHAR(255) COMMENT '页码范围',
    `source_block_ids`    TEXT COMMENT '源块ID列表',
    `create_time`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`             TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_plan_id` (`plan_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档父块表';

-- 11. 文档画像表
CREATE TABLE IF NOT EXISTS `document_profile`
(
    `id`                     BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`            BIGINT   NOT NULL COMMENT '文档ID',
    `profile_version`        INT      NOT NULL DEFAULT 0 COMMENT '配置版本',
    `document_summary`       TEXT COMMENT '文档摘要',
    `document_type`          VARCHAR(64) COMMENT '文档类型',
    `core_topics`            TEXT COMMENT '核心主题',
    `example_questions`      TEXT COMMENT '示例问题',
    `graph_friendly`         INT      NOT NULL DEFAULT 0 COMMENT '图谱友好标记',
    `supports_graph_outline` INT      NOT NULL DEFAULT 0 COMMENT '支持图谱大纲',
    `supports_item_lookup`   INT      NOT NULL DEFAULT 0 COMMENT '支持条目查询',
    `supports_graph_assist`  INT      NOT NULL DEFAULT 0 COMMENT '支持图谱辅助',
    `profile_source`         VARCHAR(255) COMMENT '配置来源',
    `profile_status`         INT      NOT NULL DEFAULT 0 COMMENT '配置状态',
    `error_msg`              TEXT COMMENT '错误信息',
    `create_time`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档画像表';

-- 12. 策略方案表
CREATE TABLE IF NOT EXISTS `document_strategy_plan`
(
    `id`                BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`       BIGINT   NOT NULL COMMENT '文档ID',
    `plan_version`      INT      NOT NULL DEFAULT 0 COMMENT '计划版本',
    `plan_source`       INT      NOT NULL DEFAULT 0 COMMENT '计划来源',
    `plan_status`       INT      NOT NULL DEFAULT 0 COMMENT '计划状态',
    `strategy_count`    INT      NOT NULL DEFAULT 0 COMMENT '策略数量',
    `strategy_snapshot` TEXT COMMENT '策略快照',
    `recommend_reason`  TEXT COMMENT '推荐理由',
    `adjust_note`       TEXT COMMENT '调整备注',
    `confirm_user_id`   BIGINT   NOT NULL DEFAULT 0 COMMENT '确认用户ID',
    `confirm_time`      DATETIME COMMENT '确认时间',
    `create_time`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`           TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='策略方案表';

-- 13. 策略步骤表
CREATE TABLE IF NOT EXISTS `document_strategy_step`
(
    `id`               BIGINT       NOT NULL COMMENT '主键ID',
    `plan_id`          BIGINT       NOT NULL COMMENT '计划ID',
    `document_id`      BIGINT       NOT NULL COMMENT '文档ID',
    `step_no`          INT          NOT NULL DEFAULT 0 COMMENT '步骤序号',
    `pipeline_type`    VARCHAR(255) NOT NULL COMMENT '管道类型',
    `strategy_type`    INT          NOT NULL DEFAULT 0 COMMENT '策略类型',
    `strategy_role`    INT          NOT NULL DEFAULT 0 COMMENT '策略角色',
    `source_type`      INT          NOT NULL DEFAULT 0 COMMENT '来源类型',
    `execute_status`   INT          NOT NULL DEFAULT 0 COMMENT '执行状态',
    `recommend_reason` TEXT COMMENT '推荐理由',
    `create_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`          TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_plan_id` (`plan_id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='策略步骤表';

-- 14. 文档结构节点表
CREATE TABLE IF NOT EXISTS `document_structure_node`
(
    `id`                    BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`           BIGINT   NOT NULL COMMENT '文档ID',
    `parse_task_id`         BIGINT   NOT NULL COMMENT '解析任务ID',
    `node_no`               INT      NOT NULL DEFAULT 0 COMMENT '节点编号',
    `node_type`             INT      NOT NULL DEFAULT 0 COMMENT '节点类型',
    `parent_node_id`        BIGINT   NOT NULL DEFAULT 0 COMMENT '父节点ID',
    `prev_sibling_node_id`  BIGINT   NOT NULL DEFAULT 0 COMMENT '前兄弟节点ID',
    `next_sibling_node_id`  BIGINT   NOT NULL DEFAULT 0 COMMENT '后兄弟节点ID',
    `depth`                 INT      NOT NULL DEFAULT 0 COMMENT '深度',
    `node_code`             VARCHAR(255) COMMENT '节点编码',
    `title`                 VARCHAR(500) COMMENT '标题',
    `anchor_text`           TEXT COMMENT '锚点文本',
    `canonical_path`        TEXT COMMENT '规范路径',
    `section_path`          TEXT COMMENT '章节路径',
    `content_text`          TEXT COMMENT '内容文本',
    `item_index`            INT      NOT NULL DEFAULT 0 COMMENT '条目索引',
    `syntax_schema_version` VARCHAR(50) COMMENT '语法结构版本',
    `syntax_source_sha256`  VARCHAR(64) COMMENT '语法源SHA256',
    `syntax_node_id`        VARCHAR(255) COMMENT '语法节点ID',
    `syntax_node_type`      VARCHAR(255) COMMENT '语法节点类型',
    `syntax_source_origin`  VARCHAR(255) COMMENT '语法源来源',
    `source_start_byte`     INT      NOT NULL DEFAULT 0 COMMENT '源起始字节',
    `source_end_byte`       INT      NOT NULL DEFAULT 0 COMMENT '源结束字节',
    `source_start_line`     INT      NOT NULL DEFAULT 0 COMMENT '源起始行',
    `source_start_column`   INT      NOT NULL DEFAULT 0 COMMENT '源起始列',
    `source_end_line`       INT      NOT NULL DEFAULT 0 COMMENT '源结束行',
    `source_end_column`     INT      NOT NULL DEFAULT 0 COMMENT '源结束列',
    `create_time`           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`               TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_parse_task_id` (`parse_task_id`),
    INDEX `idx_parent_node_id` (`parent_node_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档结构节点表';

-- 15. 文档表格表
CREATE TABLE IF NOT EXISTS `document_table`
(
    `id`            BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`   BIGINT   NOT NULL COMMENT '文档ID',
    `task_id`       BIGINT   NOT NULL COMMENT '任务ID',
    `block_id`      BIGINT   NOT NULL COMMENT '区块ID',
    `table_no`      INT      NOT NULL DEFAULT 0 COMMENT '表格编号',
    `section_path`  TEXT COMMENT '章节路径',
    `page_no`       INT      NOT NULL DEFAULT 0 COMMENT '页码',
    `page_range`    VARCHAR(50) COMMENT '页面范围',
    `bbox_json`     TEXT COMMENT '边界框JSON',
    `title`         TEXT COMMENT '标题',
    `row_count`     INT      NOT NULL DEFAULT 0 COMMENT '行数',
    `column_count`  INT      NOT NULL DEFAULT 0 COMMENT '列数',
    `table_html`    TEXT COMMENT '表格HTML',
    `metadata_json` TEXT COMMENT '元数据JSON',
    `create_time`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`       TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_block_id` (`block_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档表格表';

-- 16. 文档表格单元格表
CREATE TABLE IF NOT EXISTS `document_table_cell`
(
    `id`               BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`      BIGINT   NOT NULL COMMENT '文档ID',
    `task_id`          BIGINT   NOT NULL COMMENT '任务ID',
    `table_id`         BIGINT   NOT NULL COMMENT '表格ID',
    `row_id`           BIGINT   NOT NULL COMMENT '行ID',
    `column_id`        BIGINT   NOT NULL COMMENT '列ID',
    `row_no`           INT      NOT NULL DEFAULT 0 COMMENT '行号',
    `column_no`        INT      NOT NULL DEFAULT 0 COMMENT '列号',
    `cell_text`        TEXT COMMENT '单元格文本',
    `numeric_value`    DECIMAL(10, 2) COMMENT '数值',
    `source_row_no`    INT      NOT NULL DEFAULT 0 COMMENT '源行号',
    `source_column_no` INT      NOT NULL DEFAULT 0 COMMENT '源列号',
    `source_cell_ref`  VARCHAR(50) COMMENT '源单元格引用',
    `bbox_json`        TEXT COMMENT '边界框JSON',
    `metadata_json`    TEXT COMMENT '元数据JSON',
    `create_time`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`          TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_table_id` (`table_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档表格单元格表';

-- 17. 文档表格列表
CREATE TABLE IF NOT EXISTS `document_table_column`
(
    `id`              BIGINT       NOT NULL COMMENT '主键ID',
    `document_id`     BIGINT       NOT NULL COMMENT '文档ID',
    `task_id`         BIGINT       NOT NULL COMMENT '任务ID',
    `table_id`        BIGINT       NOT NULL COMMENT '表格ID',
    `column_no`       INT          NOT NULL DEFAULT 0 COMMENT '列编号',
    `column_name`     VARCHAR(255) NOT NULL COMMENT '列名称',
    `normalized_name` VARCHAR(255) COMMENT '标准化列名',
    `value_type`      VARCHAR(50) COMMENT '值类型',
    `create_time`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`         TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_table_id` (`table_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档表格列表';

-- 18. 文档表格行表
CREATE TABLE IF NOT EXISTS `document_table_row`
(
    `id`          BIGINT   NOT NULL COMMENT '主键ID',
    `document_id` BIGINT   NOT NULL COMMENT '文档ID',
    `task_id`     BIGINT   NOT NULL COMMENT '任务ID',
    `table_id`    BIGINT   NOT NULL COMMENT '表格ID',
    `row_no`      INT      NOT NULL DEFAULT 0 COMMENT '行号',
    `row_text`    TEXT COMMENT '行文本',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_table_id` (`table_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档表格行表';

-- 19. 文档任务表
CREATE TABLE IF NOT EXISTS `document_task`
(
    `id`                   BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`          BIGINT   NOT NULL COMMENT '文档ID',
    `plan_id`              BIGINT   NOT NULL COMMENT '计划ID',
    `source_parse_task_id` BIGINT   NOT NULL DEFAULT 0 COMMENT '源解析任务ID',
    `task_type`            INT      NOT NULL DEFAULT 0 COMMENT '任务类型',
    `task_status`          INT      NOT NULL DEFAULT 0 COMMENT '任务状态',
    `current_stage`        INT      NOT NULL DEFAULT 0 COMMENT '当前阶段',
    `trigger_source`       INT      NOT NULL DEFAULT 0 COMMENT '触发来源',
    `strategy_snapshot`    TEXT COMMENT '策略快照',
    `retry_count`          INT      NOT NULL DEFAULT 0 COMMENT '重试次数',
    `start_time`           DATETIME COMMENT '开始时间',
    `finish_time`          DATETIME COMMENT '完成时间',
    `cost_millis`          BIGINT   NOT NULL DEFAULT 0 COMMENT '耗时毫秒',
    `error_code`           VARCHAR(50) COMMENT '错误码',
    `error_msg`            TEXT COMMENT '错误信息',
    `ext_json`             TEXT COMMENT '扩展JSON',
    `create_time`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`              TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`),
    INDEX `idx_plan_id` (`plan_id`),
    INDEX `idx_task_status` (`task_status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档任务表';

-- 20. 文档任务日志表
CREATE TABLE IF NOT EXISTS `document_task_log`
(
    `id`            BIGINT   NOT NULL COMMENT '主键ID',
    `task_id`       BIGINT   NOT NULL COMMENT '任务ID',
    `document_id`   BIGINT   NOT NULL COMMENT '文档ID',
    `stage_type`    INT      NOT NULL DEFAULT 0 COMMENT '阶段类型',
    `event_type`    INT      NOT NULL DEFAULT 0 COMMENT '事件类型',
    `log_level`     INT      NOT NULL DEFAULT 0 COMMENT '日志级别',
    `operator_type` INT      NOT NULL DEFAULT 0 COMMENT '操作人类型',
    `operator_id`   BIGINT   NOT NULL DEFAULT 0 COMMENT '操作人ID',
    `content`       TEXT     NOT NULL COMMENT '内容',
    `detail_json`   TEXT COMMENT '详情JSON',
    `create_time`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`       TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档任务日志表';

-- 21. 知识库表
CREATE TABLE IF NOT EXISTS `knowledge_base`
(
    `id`                    BIGINT       NOT NULL COMMENT '主键ID',
    `base_name`             VARCHAR(255) NOT NULL COMMENT '基础名称',
    `description`           TEXT COMMENT '描述',
    `embedding_model`       VARCHAR(100) NOT NULL COMMENT '嵌入模型',
    `retrieval_config_json` JSON COMMENT '检索配置JSON',
    `graph_rag_config_json` JSON COMMENT '图谱RAG配置JSON',
    `raptor_config_json`    JSON COMMENT 'Raptor配置JSON',
    `metadata_filter_json`  JSON COMMENT '元数据过滤JSON',
    `is_default`            TINYINT COMMENT '是否默认',
    `sort_order`            INT          NOT NULL DEFAULT 0 COMMENT '排序顺序',
    `create_time`           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`               TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_base_name` (`base_name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='知识库表';

-- 22. 文档画像（知识库）表
CREATE TABLE IF NOT EXISTS `knowledge_document_profile`
(
    `id`                     BIGINT   NOT NULL COMMENT '主键ID',
    `document_id`            BIGINT   NOT NULL COMMENT '文档ID',
    `profile_version`        INT      NOT NULL DEFAULT 0 COMMENT '画像版本',
    `document_summary`       TEXT COMMENT '文档摘要',
    `document_type`          VARCHAR(255) COMMENT '文档类型',
    `core_topics`            TEXT COMMENT '核心主题',
    `example_questions`      TEXT COMMENT '示例问题',
    `graph_friendly`         INT      NOT NULL DEFAULT 0 COMMENT '是否支持图表示',
    `supports_graph_outline` INT      NOT NULL DEFAULT 0 COMMENT '是否支持图大纲',
    `supports_item_lookup`   INT      NOT NULL DEFAULT 0 COMMENT '是否支持项目查找',
    `supports_graph_assist`  INT      NOT NULL DEFAULT 0 COMMENT '是否支持图辅助',
    `profile_source`         VARCHAR(255) COMMENT '画像来源',
    `profile_status`         INT      NOT NULL DEFAULT 0 COMMENT '画像状态',
    `error_msg`              TEXT COMMENT '错误信息',
    `create_time`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                TINYINT  NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='文档画像（知识库）表';

-- 23. 知识路由追踪表
CREATE TABLE IF NOT EXISTS `knowledge_route_trace`
(
    `id`                                 BIGINT         NOT NULL COMMENT '主键ID',
    `conversation_id`                    VARCHAR(255)   NOT NULL COMMENT '会话ID',
    `exchange_id`                        BIGINT         NOT NULL COMMENT '交换ID',
    `question`                           TEXT           NOT NULL COMMENT '原始问题',
    `rewrite_question`                   TEXT COMMENT '改写后问题',
    `mode`                               VARCHAR(50)    NOT NULL COMMENT '路由模式',
    `knowledge_base_selection_mode`      VARCHAR(50)    NOT NULL COMMENT '知识库选择模式',
    `selected_knowledge_base_ids_json`   TEXT COMMENT '选中知识库ID列表JSON',
    `selected_knowledge_base_names_json` TEXT COMMENT '选中知识库名称列表JSON',
    `allowed_document_ids_json`          TEXT COMMENT '允许文档ID列表JSON',
    `top_scopes_json`                    TEXT COMMENT '顶级Scope列表JSON',
    `top_topics_json`                    TEXT COMMENT '顶级Topic列表JSON',
    `top_documents_json`                 TEXT COMMENT '顶级文档列表JSON',
    `selected_document_id`               BIGINT         NOT NULL DEFAULT 0 COMMENT '选中文档ID',
    `hit_selected_document`              TINYINT COMMENT '是否命中选中文档',
    `confidence`                         DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '置信度',
    `route_status`                       TINYINT        NOT NULL DEFAULT 0 COMMENT '路由状态',
    `error_msg`                          TEXT COMMENT '错误信息',
    `create_time`                        DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`                        DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                            TINYINT        NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_conversation_id` (`conversation_id`),
    INDEX `idx_exchange_id` (`exchange_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='知识路由追踪表';

-- 24. 知识范围节点表
CREATE TABLE IF NOT EXISTS `knowledge_scope_node`
(
    `id`                BIGINT       NOT NULL COMMENT '主键ID',
    `knowledge_base_id` BIGINT       NOT NULL COMMENT '知识库ID',
    `scope_name`        VARCHAR(255) NOT NULL COMMENT '范围名称',
    `parent_scope_id`   BIGINT       NOT NULL DEFAULT 0 COMMENT '父范围ID',
    `description`       TEXT COMMENT '描述',
    `aliases`           TEXT COMMENT '别名',
    `examples`          TEXT COMMENT '示例',
    `sort_order`        INT          NOT NULL DEFAULT 0 COMMENT '排序序号',
    `create_time`       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`           TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_knowledge_base_id` (`knowledge_base_id`),
    INDEX `idx_parent_scope_id` (`parent_scope_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='知识范围节点表';

-- 25. 主题-文档映射关系表
CREATE TABLE IF NOT EXISTS `knowledge_topic_document_relation`
(
    `id`                BIGINT         NOT NULL COMMENT '主键ID',
    `knowledge_base_id` BIGINT         NOT NULL COMMENT '所属知识库ID',
    `topic_id`          BIGINT         NOT NULL COMMENT '关联Topic ID',
    `document_id`       BIGINT         NOT NULL COMMENT '关联文档ID',
    `relation_score`    DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '关联置信度',
    `relation_source`   VARCHAR(255)   NOT NULL COMMENT '关系来源',
    `reason`            TEXT COMMENT '关联原因',
    `create_time`       DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`       DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`           TINYINT        NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_knowledge_base_id` (`knowledge_base_id`),
    INDEX `idx_topic_id` (`topic_id`),
    INDEX `idx_document_id` (`document_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='主题-文档映射关系表';

-- 26. 知识话题节点表
CREATE TABLE IF NOT EXISTS `knowledge_topic_node`
(
    `id`                   BIGINT       NOT NULL COMMENT '主键ID',
    `knowledge_base_id`    BIGINT       NOT NULL COMMENT '知识库ID',
    `topic_name`           VARCHAR(255) NOT NULL COMMENT '主题名称',
    `scope_id`             BIGINT       NOT NULL DEFAULT 0 COMMENT '范围ID',
    `description`          TEXT COMMENT '描述',
    `aliases`              TEXT COMMENT '别名',
    `examples`             TEXT COMMENT '示例',
    `answer_shape`         TEXT COMMENT '答案结构',
    `execution_preference` VARCHAR(50)  NOT NULL DEFAULT '' COMMENT '执行偏好',
    `sort_order`           INT          NOT NULL DEFAULT 0 COMMENT '排序顺序',
    `create_time`          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`              TINYINT      NOT NULL DEFAULT 1 COMMENT '删除标记(1:未删除,0:已删除)',
    PRIMARY KEY (`id`),
    INDEX `idx_knowledge_base_id` (`knowledge_base_id`),
    INDEX `idx_scope_id` (`scope_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='知识话题节点表';