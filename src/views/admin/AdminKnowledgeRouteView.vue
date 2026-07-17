<template>
  <section class="knowledge-page">
    <header class="page-header">
      <div>
        <span class="section-eyebrow">Knowledge Routing</span>
        <h3>知识路由配置</h3>
        <p>按 范围 → 主题 → 画像 → 关联 的顺序逐步配置，构建自动知识问答的候选预选体系。</p>
      </div>
      <div class="header-actions">
        <button :disabled="loading || actionLoading" class="ghost-button" type="button"
                @click="loadAll">刷新数据
        </button>
        <button :disabled="!documents.length || batchLoading" class="primary-button" type="button"
                @click="regenerateAllProfiles">
          {{ batchLoading ? '批量重建中...' : '批量重建画像' }}
        </button>
      </div>
    </header>

    <div v-if="notice.message" :class="`page-notice-${notice.type}`" class="page-notice">
      {{ notice.message }}
    </div>

    <section class="stats-grid">
      <article v-for="item in summaryCards" :key="item.label" class="stat-card">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
        <small>{{ item.description }}</small>
      </article>
    </section>

    <section class="coverage-panel">
      <div class="section-card-head" style="cursor:pointer"
           @click="coveragePanelCollapsed = !coveragePanelCollapsed">
        <div>
          <span class="section-eyebrow">Scope Coverage</span>
          <h4>范围覆盖率统计</h4>
        </div>
        <div class="coverage-toggle-row">
          <span class="helper-pill helper-pill-soft">整体覆盖率 {{ overallCoverageRateText }}</span>
          <span :class="{ collapsed: coveragePanelCollapsed }" class="collapse-arrow">&#9660;</span>
        </div>
      </div>
      <div v-show="!coveragePanelCollapsed" class="coverage-grid">
        <article v-for="item in scopeCoverageRows" :key="item.scopeCode" :class="{ warning: item.pendingTopicCount > 0 }"
                 class="coverage-card">
          <div class="coverage-head">
            <div>
              <strong>{{ item.scopeName }}</strong>
              <span>{{ item.scopeCode }}</span>
            </div>
            <span class="coverage-rate">{{ item.coverageRateText }}</span>
          </div>
          <div class="coverage-track"><span :style="{ width: item.coverageRateText }"></span></div>
          <div class="coverage-metrics">
            <span>主题 {{ item.topicCount }}</span>
            <span>已覆盖 {{ item.coveredTopicCount }}</span>
            <span>未关联 {{ item.pendingTopicCount }}</span>
            <span>文档 {{ item.documentCount }}</span>
          </div>
        </article>
      </div>
    </section>

    <!-- Tab Navigation -->
    <nav class="tab-nav">
      <button v-for="tab in TAB_LIST" :key="tab.key" :class="{ active: activeTab === tab.key }"
              class="tab-btn"
              type="button" @click="activeTab = tab.key">
        <span class="tab-step">{{ tab.step }}</span>
        <span class="tab-label">{{ tab.label }}</span>
        <span class="tab-hint">{{ tab.hint }}</span>
      </button>
    </nav>

    <!-- Tab 1: 知识范围 -->
    <section v-show="activeTab === 'scope'" class="tab-content">
      <article class="panel-card">
        <div class="panel-head">
          <div>
            <h4>知识范围</h4>
            <p>先把大范围定清楚，自动知识问答才能稳定地在正确文档池里预选。</p>
          </div>
          <button class="primary-button" type="button" @click="openCreateDrawer('scope')">新建范围
          </button>
        </div>
        <div class="toolbar-row">
          <input v-model.trim="scopeKeyword" placeholder="按范围编码、名称或描述筛选"/>
        </div>
        <div class="card-grid">
          <article v-for="item in filteredScopes" :key="item.scopeCode" :class="{ active: item.scopeCode === activeScopeCode }"
                   class="data-card"
                   @click="openDrawer('scope', item, 'view')">
            <div class="data-card-head">
              <strong>{{ item.scopeName }}</strong>
            </div>
            <small>{{ item.description || '暂无描述' }}</small>
            <div class="data-card-meta">
              <span>主题 {{ topics.filter(t => t.scopeCode === item.scopeCode).length }}</span>
              <span>文档 {{
                  documents.filter(d => d.knowledgeScopeCode === item.scopeCode).length
                }}</span>
            </div>
          </article>
          <div v-if="!filteredScopes.length" class="empty-inline">没有匹配的知识范围。</div>
        </div>
      </article>
    </section>

    <!-- Tab 2: 知识主题 -->
    <section v-show="activeTab === 'topic'" class="tab-content">
      <article class="panel-card">
        <div class="panel-head">
          <div>
            <h4>知识主题</h4>
            <p>主题是范围里的可回答单元，后续会通过主题文档关联把文档候选进一步收窄。</p>
          </div>
          <button class="primary-button" type="button" @click="openCreateDrawer('topic')">新建主题
          </button>
        </div>
        <div class="toolbar-row toolbar-row-filters">
          <select v-model="activeScopeCode" class="filter-select">
            <option value="">全部范围</option>
            <option v-for="item in scopes" :key="item.scopeCode" :value="item.scopeCode">
              {{ item.scopeName }}
            </option>
          </select>
          <input v-model.trim="topicKeyword" placeholder="按主题编码、名称、别名或描述筛选"/>
        </div>
        <div class="card-grid">
          <article v-for="item in filteredTopics" :key="item.topicCode" :class="{ active: item.topicCode === activeTopicCode }"
                   class="data-card"
                   @click="openDrawer('topic', item, 'view')">
            <div class="data-card-head">
              <strong>{{ item.topicName }}</strong>
            </div>
            <div class="topic-meta-row">
              <span class="tag-chip tag-chip-soft">{{
                  formatAnswerShapeLabel(item.answerShape)
                }}</span>
              <span
                class="tag-chip tag-chip-soft">{{
                  formatExecutionPreferenceLabel(item.executionPreference)
                }}</span>
            </div>
            <small>{{ item.description || '暂无描述' }}</small>
          </article>
          <div v-if="!filteredTopics.length" class="empty-inline">当前范围下还没有主题。</div>
        </div>
      </article>
    </section>

    <!-- Tab 3: 文档画像 -->
    <section v-show="activeTab === 'profile'" class="tab-content">
      <article class="panel-card">
        <div class="panel-head">
          <div>
            <h4>文档画像</h4>
            <p>查看文档的类型、摘要、核心主题和图能力开关，判断自动路由是否有足够信息。</p>
          </div>
        </div>
        <div class="toolbar-row">
          <input v-model.trim="documentKeyword"
                 placeholder="按文档名、范围、业务分类或标签筛选文档"/>
        </div>

        <section v-if="profileAnomalyRows.length" class="anomaly-panel">
          <div class="section-card-head" style="cursor:pointer"
               @click="anomalyCollapsed = !anomalyCollapsed">
            <div>
              <span class="section-eyebrow">Profile Anomalies</span>
              <h4>画像异常清单 ({{ profileAnomalyRows.length }})</h4>
            </div>
            <div class="coverage-toggle-row">
              <button :disabled="!selectedProfileRepairIds.length || batchLoading" class="ghost-button"
                      type="button"
                      @click.stop="batchRepairProfiles">
                {{ batchLoading ? '修复中...' : `批量重建 ${selectedProfileRepairIds.length} 份` }}
              </button>
              <span :class="{ collapsed: anomalyCollapsed }" class="collapse-arrow">&#9660;</span>
            </div>
          </div>
          <div v-show="!anomalyCollapsed">
            <div class="batch-actions">
              <label class="toggle-chip">
                <input :checked="allVisibleAnomaliesSelected" type="checkbox"
                       @change="toggleAllVisibleAnomalies"/>
                <span>全选异常</span>
              </label>
            </div>
            <div class="anomaly-list">
              <article v-for="item in profileAnomalyRows" :key="`anomaly-${item.documentId}`"
                       :class="item.tone"
                       class="anomaly-card">
                <label class="row-check">
                  <input :checked="selectedProfileRepairIds.includes(item.documentId)"
                         type="checkbox"
                         @change="toggleProfileRepair(item.documentId)"/>
                  <span></span>
                </label>
                <div class="anomaly-main">
                  <strong>{{ item.documentName }}</strong>
                  <span>{{ item.scopeText }}</span>
                  <div class="tag-list">
                    <span v-for="problem in item.problems" :key="`${item.documentId}-${problem}`"
                          class="tag-chip tag-chip-warning">{{ problem }}</span>
                  </div>
                </div>
                <button class="ghost-button" type="button"
                        @click.stop="selectAnomalyDocument(item); openDrawer('profile', null, 'view')">
                  查看
                </button>
              </article>
            </div>
          </div>
        </section>

        <div class="card-grid">
          <article v-for="item in filteredDocuments" :key="item.documentId" :class="{ active: item.documentId === profileDocumentId }"
                   class="data-card"
                   @click="selectDocument(item); openDrawer('profile', null, 'view')">
            <div class="data-card-head">
              <strong>{{ item.documentName }}</strong>
            </div>
            <small>{{ documentMetaLine(item) }}</small>
          </article>
          <div v-if="!filteredDocuments.length" class="empty-inline">没有匹配的文档。</div>
        </div>
      </article>
    </section>

    <!-- Tab 4: 主题文档关联 -->
    <section v-show="activeTab === 'relation'" class="tab-content">
      <article class="panel-card">
        <div class="panel-head">
          <div>
            <h4>主题文档关联</h4>
            <p>把"哪个主题该优先看哪份文档"显式维护下来，低置信自动路由时会直接受益。</p>
          </div>
          <button class="primary-button" type="button" @click="openCreateDrawer('relation')">
            新建关联
          </button>
        </div>
        <div class="toolbar-row toolbar-row-filters">
          <select v-model="activeScopeCode" class="filter-select">
            <option value="">全部范围</option>
            <option v-for="item in scopes" :key="item.scopeCode" :value="item.scopeCode">
              {{ item.scopeName }}
            </option>
          </select>
          <input v-model.trim="relationKeyword" placeholder="按主题、文档、原因筛选关联结果"/>
          <button :disabled="actionLoading" class="ghost-button" type="button"
                  @click="loadRelations">刷新
          </button>
        </div>
        <div class="helper-bar">
          <span class="helper-pill helper-pill-soft">{{ relations.length }} 条可见关联</span>
        </div>
        <div class="relation-table">
          <article v-for="item in relations" :key="`${item.topicCode}-${item.documentId}`"
                   class="relation-row"
                   @click="openDrawer('relation', item, 'view')">
            <div>
              <strong>{{ item.documentName }}</strong>
              <span>{{ item.topicCode }} · 分数 {{
                  item.relationScore
                }} · {{
                  item.knowledgeScopeName ||
                  item.knowledgeScopeCode || '未分范围'
                }}</span>
              <small>{{ item.reason || documentMetaLine(item) }}</small>
            </div>
            <button :disabled="actionLoading" class="danger-link" type="button"
                    @click.stop="removeRelation(item)">移除
            </button>
          </article>
          <div v-if="!relations.length" class="empty-inline">当前筛选下还没有保存的文档关联。</div>
        </div>
      </article>
    </section>

    <!-- Drawer -->
    <transition name="drawer-fade">
      <div v-if="drawerVisible" class="drawer-overlay" @click="closeDrawer"></div>
    </transition>
    <transition name="drawer-slide">
      <aside v-if="drawerVisible" class="drawer-panel">
        <div class="drawer-header">
          <h4 v-if="drawerType === 'scope'">
            {{ drawerMode === 'edit' && !scopeTarget ? '新建知识范围' : '知识范围详情' }}</h4>
          <h4 v-else-if="drawerType === 'topic'">
            {{ drawerMode === 'edit' && !topicTarget ? '新建知识主题' : '知识主题详情' }}</h4>
          <h4 v-else-if="drawerType === 'profile'">文档画像详情</h4>
          <h4 v-else-if="drawerType === 'relation'">
            {{ drawerMode === 'edit' && !relationTarget ? '新建主题文档关联' : '关联详情' }}
          </h4>
          <button class="ghost-button drawer-close" type="button" @click="closeDrawer">关闭</button>
        </div>
        <div class="drawer-body">

          <!-- Scope Drawer -->
          <template v-if="drawerType === 'scope'">
            <template v-if="drawerMode === 'view' && scopeTarget">
              <div class="drawer-detail">
                <div class="detail-row">
                  <span>范围编码</span>
                  <strong>{{ scopeTarget.scopeCode }}</strong>
                </div>
                <div class="detail-row">
                  <span>范围名称</span>
                  <strong>{{ scopeTarget.scopeName }}</strong>
                </div>
                <div class="detail-row">
                  <span>父级编码</span>
                  <strong>{{ scopeTarget.parentScopeCode || '-' }}</strong>
                </div>
                <div class="detail-row">
                  <span>排序值</span>
                  <strong>{{ scopeTarget.sortOrder }}</strong>
                </div>
                <div class="detail-row">
                  <span>描述</span>
                  <p>{{ scopeTarget.description || '暂无描述' }}</p>
                </div>
                <div v-if="scopeTarget.aliases" class="tag-section">
                  <p>别名</p>
                  <div class="tag-list">
                    <span v-for="a in parseTextList(scopeTarget.aliases)" :key="a"
                          class="tag-chip tag-chip-soft">{{
                        a
                      }}</span>
                  </div>
                </div>
                <div v-if="scopeTarget.examples" class="tag-section">
                  <p>典型问题</p>
                  <div class="tag-list">
                    <span v-for="e in parseJsonArray(scopeTarget.examples)" :key="e"
                          class="tag-chip">{{
                        e
                      }}</span>
                  </div>
                </div>
              </div>
            </template>
            <template v-if="drawerMode === 'edit'">
              <div class="form-grid">
                <input v-model="scopeForm.scopeCode" placeholder="范围编码，例如 operation_rule"/>
                <input v-model="scopeForm.scopeName" placeholder="范围名称，例如 运营规则"/>
                <input v-model="scopeForm.parentScopeCode" placeholder="父级编码，可空"/>
                <input v-model="scopeForm.aliases" placeholder="别名，英文逗号分隔"/>
                <input v-model="scopeForm.sortOrder" placeholder="排序值"/>
                <textarea v-model="scopeForm.description" placeholder="范围描述"></textarea>
                <textarea v-model="scopeForm.examples"
                          placeholder='典型问题 JSON，例如 ["上线观察多久"]'></textarea>
              </div>
            </template>
          </template>

          <!-- Topic Drawer -->
          <template v-if="drawerType === 'topic'">
            <template v-if="drawerMode === 'view' && topicTarget">
              <div class="drawer-detail">
                <div class="detail-row">
                  <span>主题编码</span>
                  <strong>{{ topicTarget.topicCode }}</strong>
                </div>
                <div class="detail-row">
                  <span>主题名称</span>
                  <strong>{{ topicTarget.topicName }}</strong>
                </div>
                <div class="detail-row">
                  <span>所属范围</span>
                  <strong>{{ topicTarget.scopeCode }}</strong>
                </div>
                <div class="detail-row">
                  <span>回答形态</span>
                  <strong>{{ formatAnswerShapeLabel(topicTarget.answerShape) }}</strong>
                </div>
                <div class="detail-row">
                  <span>执行偏好</span>
                  <strong>{{
                      formatExecutionPreferenceLabel(topicTarget.executionPreference)
                    }}</strong>
                </div>
                <div class="detail-row">
                  <span>排序值</span>
                  <strong>{{ topicTarget.sortOrder }}</strong>
                </div>
                <div class="detail-row">
                  <span>描述</span>
                  <p>{{ topicTarget.description || '暂无描述' }}</p>
                </div>
                <div v-if="topicTarget.aliases" class="tag-section">
                  <p>别名</p>
                  <div class="tag-list">
                    <span v-for="a in parseTextList(topicTarget.aliases)" :key="a"
                          class="tag-chip tag-chip-soft">{{
                        a
                      }}</span>
                  </div>
                </div>
                <div v-if="topicTarget.examples" class="tag-section">
                  <p>典型问题</p>
                  <div class="tag-list">
                    <span v-for="e in parseJsonArray(topicTarget.examples)" :key="e"
                          class="tag-chip">{{ e }}</span>
                  </div>
                </div>
              </div>
            </template>
            <template v-if="drawerMode === 'edit'">
              <div class="form-grid">
                <input v-model="topicForm.topicCode" placeholder="主题编码"/>
                <input v-model="topicForm.topicName" placeholder="主题名称"/>
                <select v-model="topicForm.scopeCode">
                  <option value="">选择所属范围</option>
                  <option v-for="item in scopes" :key="item.scopeCode" :value="item.scopeCode">
                    {{ item.scopeName }}
                  </option>
                </select>
                <input v-model="topicForm.aliases" placeholder="别名，英文逗号分隔"/>
                <select v-model="topicForm.answerShape">
                  <option value="">选择回答形态</option>
                  <option v-for="item in ANSWER_SHAPE_OPTIONS" :key="item.value"
                          :value="item.value">{{ item.label }}
                  </option>
                </select>
                <select v-model="topicForm.executionPreference">
                  <option value="">选择执行偏好</option>
                  <option v-for="item in EXECUTION_PREFERENCE_OPTIONS" :key="item.value"
                          :value="item.value">{{
                      item.label
                    }}
                  </option>
                </select>
                <input v-model="topicForm.sortOrder" placeholder="排序值"/>
                <textarea v-model="topicForm.description" placeholder="主题描述"></textarea>
                <textarea v-model="topicForm.examples" placeholder='典型问题 JSON'></textarea>
              </div>
            </template>
          </template>

          <!-- Profile Drawer -->
          <template v-if="drawerType === 'profile'">
            <div class="drawer-detail">
              <div class="detail-row">
                <span>文档名称</span>
                <strong>{{ selectedProfileDocument?.documentName || '-' }}</strong>
              </div>
              <div class="detail-row">
                <span>元数据</span>
                <small>{{ selectedProfileDocumentMeta }}</small>
              </div>
            </div>
            <div class="actions" style="margin-top:12px">
              <button :disabled="!profileDocumentId || actionLoading" class="primary-button"
                      type="button"
                      @click="loadProfile">查看画像
              </button>
              <button :disabled="!profileDocumentId || actionLoading" class="ghost-button"
                      type="button"
                      @click="regenerateProfile">重新生成
              </button>
            </div>
            <div v-if="profile" class="profile-card" style="margin-top:16px">
              <div class="profile-head">
                <strong>{{
                    selectedProfileDocument?.documentName || `文档 ${profileDocumentId}`
                  }}</strong>
                <span :class="profileStatusClass(profile.profileStatus)"
                      class="profile-status-pill">{{
                    profileStatusText(profile.profileStatus)
                  }}</span>
              </div>
              <p class="profile-summary">{{
                  profile.documentSummary || '当前画像还没有生成摘要。'
                }}</p>
              <div class="profile-grid">
                <article class="mini-card">
                  <span>文档类型</span>
                  <strong>{{ formatDocumentTypeLabel(profile.documentType) }}</strong>
                </article>
                <article class="mini-card">
                  <span>画像来源</span>
                  <strong>{{ formatProfileSourceLabel(profile.profileSource) }}</strong>
                </article>
                <article class="mini-card">
                  <span>图能力</span>
                  <strong>{{ graphCapabilityText(profile) }}</strong>
                </article>
                <article class="mini-card">
                  <span>核心主题数</span>
                  <strong>{{ parseJsonArray(profile.coreTopics).length }}</strong>
                </article>
              </div>
              <div class="tag-section">
                <p>核心主题</p>
                <div class="tag-list">
                  <span v-for="item in parseJsonArray(profile.coreTopics)" :key="`dt-${item}`"
                        class="tag-chip">{{
                      item
                    }}</span>
                  <span v-if="!parseJsonArray(profile.coreTopics).length"
                        class="tag-chip tag-chip-empty">暂无</span>
                </div>
              </div>
              <div class="tag-section">
                <p>示例问题</p>
                <div class="tag-list">
                  <span v-for="item in parseJsonArray(profile.exampleQuestions)" :key="`dq-${item}`"
                        class="tag-chip tag-chip-soft">{{ item }}</span>
                  <span v-if="!parseJsonArray(profile.exampleQuestions).length"
                        class="tag-chip tag-chip-empty">暂无</span>
                </div>
              </div>
            </div>
          </template>

          <!-- Relation Drawer -->
          <template v-if="drawerType === 'relation'">
            <template v-if="drawerMode === 'view' && relationTarget">
              <div class="drawer-detail">
                <div class="detail-row">
                  <span>主题编码</span>
                  <strong>{{ relationTarget.topicCode }}</strong>
                </div>
                <div class="detail-row">
                  <span>文档名称</span>
                  <strong>{{ relationTarget.documentName }}</strong>
                </div>
                <div class="detail-row">
                  <span>关联分数</span>
                  <strong>{{ relationTarget.relationScore }}</strong>
                </div>
                <div class="detail-row">
                  <span>关联来源</span>
                  <strong>{{ relationTarget.relationSource || '-' }}</strong>
                </div>
                <div class="detail-row">
                  <span>原因</span>
                  <p>{{ relationTarget.reason || '未填写' }}</p>
                </div>
              </div>
            </template>
            <template v-if="drawerMode === 'edit'">
              <div class="form-grid">
                <select v-model="relationForm.topicCode">
                  <option value="">选择主题</option>
                  <option v-for="item in topics" :key="item.topicCode" :value="item.topicCode">
                    {{ item.topicName }}
                  </option>
                </select>
                <select v-model="relationForm.documentId">
                  <option value="">选择文档</option>
                  <option v-for="item in documents" :key="item.documentId" :value="item.documentId">
                    {{ item.documentName }}
                  </option>
                </select>
                <input v-model="relationForm.relationScore" placeholder="关联分数，例如 0.9200"/>
                <input v-model="relationForm.reason" placeholder="关联原因"/>
              </div>
            </template>
          </template>
        </div>

        <div class="drawer-footer">
          <template v-if="drawerType === 'scope'">
            <template v-if="drawerMode === 'view'">
              <button class="primary-button" type="button" @click="switchDrawerToEdit">编辑</button>
              <button :disabled="actionLoading" class="ghost-button ghost-danger" type="button"
                      @click="deleteScope">删除
              </button>
            </template>
            <template v-else>
              <button :disabled="actionLoading" class="primary-button" type="button"
                      @click="saveScope">保存
              </button>
              <button class="ghost-button" type="button" @click="closeDrawer">取消</button>
            </template>
          </template>
          <template v-if="drawerType === 'topic'">
            <template v-if="drawerMode === 'view'">
              <button class="primary-button" type="button" @click="switchDrawerToEdit">编辑</button>
              <button :disabled="actionLoading" class="ghost-button ghost-danger" type="button"
                      @click="deleteTopic">删除
              </button>
            </template>
            <template v-else>
              <button :disabled="actionLoading" class="primary-button" type="button"
                      @click="saveTopic">保存
              </button>
              <button class="ghost-button" type="button" @click="closeDrawer">取消</button>
            </template>
          </template>
          <template v-if="drawerType === 'profile'">
            <button class="ghost-button" type="button" @click="closeDrawer">关闭</button>
          </template>
          <template v-if="drawerType === 'relation'">
            <template v-if="drawerMode === 'view'">
              <button class="primary-button" type="button" @click="switchDrawerToEdit">编辑</button>
              <button :disabled="actionLoading" class="ghost-button ghost-danger" type="button"
                      @click="removeRelation(relationTarget); closeDrawer()">移除
              </button>
            </template>
            <template v-else>
              <button :disabled="actionLoading" class="primary-button" type="button"
                      @click="saveRelation">保存
              </button>
              <button class="ghost-button" type="button" @click="closeDrawer">取消</button>
            </template>
          </template>
        </div>
      </aside>
    </transition>
  </section>
</template>

<script lang="ts" setup>
import {computed, onMounted, reactive, ref, watch} from 'vue'
import {knowledgeApi} from '@/api/knowledge'
import type {
  DocumentDetailResp,
  DocumentProfileResp,
  KnowledgeScopeResp,
  KnowledgeScopeSaveReq,
  KnowledgeTopicResp,
  KnowledgeTopicSaveReq,
  TopicDocumentRelationResp,
  TopicDocumentRelationSaveReq
} from '@/types'
import {documentApi} from '@/api/document.ts'

const OPERATOR_ID = '10001'
const ANSWER_SHAPE_OPTIONS = Object.freeze([
  {value: 'list', label: '列表型回答'},
  {value: 'explain', label: '解释说明型回答'},
  {value: 'steps', label: '步骤型回答'}
])
const EXECUTION_PREFERENCE_OPTIONS = Object.freeze([
  {value: 'retrieval', label: '普通检索优先'},
  {value: 'graph_assist', label: '图辅助优先'}
])
const DOCUMENT_TYPE_OPTIONS = Object.freeze([
  {value: 'intro', label: '介绍型文档'},
  {value: 'manual', label: '操作手册'},
  {value: 'rule', label: '规则文档'},
  {value: 'faq', label: '常见问题'},
  {value: 'troubleshooting', label: '故障排查'},
  {value: 'spec', label: '规格说明'}
])
const PROFILE_SOURCE_OPTIONS = Object.freeze([
  {value: 'auto', label: '自动生成'},
  {value: 'manual', label: '手动维护'},
  {value: 'mixed', label: '自动 + 手动'}
])
const ANSWER_SHAPE_LABEL_MAP = Object.freeze(
  Object.fromEntries(ANSWER_SHAPE_OPTIONS.map(item => [item.value, item.label]))
)
const EXECUTION_PREFERENCE_LABEL_MAP = Object.freeze(
  Object.fromEntries(EXECUTION_PREFERENCE_OPTIONS.map(item => [item.value, item.label]))
)
const DOCUMENT_TYPE_LABEL_MAP = Object.freeze(
  Object.fromEntries(DOCUMENT_TYPE_OPTIONS.map(item => [item.value, item.label]))
)
const PROFILE_SOURCE_LABEL_MAP = Object.freeze(
  Object.fromEntries(PROFILE_SOURCE_OPTIONS.map(item => [item.value, item.label]))
)
const TAB_LIST = Object.freeze<TabItem[]>([
  {key: 'scope', label: '知识范围', step: 1, hint: '定义知识领域边界'},
  {key: 'topic', label: '知识主题', step: 2, hint: '范围下的可回答单元'},
  {key: 'profile', label: '文档画像', step: 3, hint: '文档类型与能力分析'},
  {key: 'relation', label: '主题文档关联', step: 4, hint: '主题与文档的绑定关系'}
])

const loading = ref<boolean>(false)
const actionLoading = ref<boolean>(false)
const batchLoading = ref<boolean>(false)
const scopes = ref<KnowledgeScopeResp[]>([])
const topics = ref<KnowledgeTopicResp[]>([])
const documents = ref<DocumentDetailResp[]>([])
const allRelations = ref<TopicDocumentRelationResp[]>([])
const profileDocumentId = ref<string>('')
const profile = ref<DocumentProfileResp | null>(null)
const activeScopeCode = ref<string>('')
const activeTopicCode = ref<string>('')
const scopeKeyword = ref<string>('')
const topicKeyword = ref<string>('')
const documentKeyword = ref<string>('')
const relationKeyword = ref<string>('')
const selectedProfileRepairIds = ref<string[]>([])
const drawerVisible = ref<boolean>(false)
const drawerMode = ref<'view' | 'edit'>('view')
const drawerType = ref<TabKey>('')
const coveragePanelCollapsed = ref<boolean>(false)
const anomalyCollapsed = ref<boolean>(true)
const scopeTarget = ref<KnowledgeScopeResp | null>(null)
const topicTarget = ref<KnowledgeTopicResp | null>(null)
const profileTarget = ref<DocumentProfileResp | null>(null)
const relationTarget = ref<TopicDocumentRelationResp | null>(null)

interface Notice {
  type: 'info' | 'success' | 'danger' | 'warning'
  message: string
}

const notice = reactive<Notice>({type: 'info', message: ''})

type TabKey = 'scope' | 'topic' | 'profile' | 'relation' | ''

interface TabItem {
  key: TabKey
  label: string
  step: number
  hint: string
}

const activeTab = ref<TabKey>('scope')

function openDrawer(type: TabKey, item: KnowledgeScopeResp | KnowledgeTopicResp | TopicDocumentRelationResp | DocumentProfileResp | null, mode: 'view' | 'edit' = 'view'): void {
  drawerType.value = type
  drawerMode.value = mode
  drawerVisible.value = true

  switch (type) {
    case 'scope':
      scopeTarget.value = item as KnowledgeScopeResp || null
      break
    case 'topic':
      topicTarget.value = item as KnowledgeTopicResp || null
      break
    case 'profile':
      profileTarget.value = item as DocumentProfileResp || null
      break
    case 'relation':
      relationTarget.value = item as TopicDocumentRelationResp || null
      break
  }
}

function closeDrawer(): void {
  drawerVisible.value = false
  scopeTarget.value = null
  topicTarget.value = null
  profileTarget.value = null
  relationTarget.value = null
  drawerMode.value = 'view'
  drawerType.value = 'scope'
}

function switchDrawerToEdit(): void {
  drawerMode.value = 'edit'
  if (drawerType.value === 'scope' && scopeTarget.value) {
    editScope(scopeTarget.value)
  } else if (drawerType.value === 'topic' && topicTarget.value) {
    editTopic(topicTarget.value)
  } else if (drawerType.value === 'relation' && relationTarget.value) {
    const target = relationTarget.value
    Object.assign(relationForm, {
      topicCode: target.topicCode || '',
      documentId: target.documentId || '',
      relationScore: target.relationScore || '0.9000',
      relationSource: 'manual',
      reason: target.reason || '',
      operatorId: OPERATOR_ID
    })
  }
}

function openCreateDrawer(type: TabKey): void {
  if (type === 'scope') {
    resetScopeForm()
  } else if (type === 'topic') {
    resetTopicForm()
  } else if (type === 'relation') {
    Object.assign(relationForm, {
      topicCode: activeTopicCode.value || '',
      documentId: '',
      relationScore: '0.9000',
      relationSource: 'manual',
      reason: '',
      operatorId: OPERATOR_ID
    })
  }
  openDrawer(type, null, 'edit')
}

const scopeForm = reactive<KnowledgeScopeSaveReq>({
  scopeCode: '',
  scopeName: '',
  parentScopeCode: '',
  description: '',
  aliases: '',
  examples: '',
  sortOrder: 0,
  operatorId: OPERATOR_ID
})

const topicForm = reactive<KnowledgeTopicSaveReq>({
  topicCode: '',
  topicName: '',
  scopeCode: '',
  description: '',
  aliases: '',
  examples: '',
  answerShape: '',
  executionPreference: '',
  sortOrder: 0,
  operatorId: OPERATOR_ID
})

const relationForm = reactive<TopicDocumentRelationSaveReq>({
  topicCode: '',
  documentId: '',
  relationScore: 0.9,
  relationSource: 'manual',
  reason: '',
  operatorId: OPERATOR_ID
})

const activeScope = computed<KnowledgeScopeResp | null>(() => scopes.value.find((item) => item.scopeCode === activeScopeCode.value) || null)
const activeTopic = computed<KnowledgeTopicResp | null>(() => topics.value.find((item) => item.topicCode === activeTopicCode.value) || null)
const filteredScopes = computed<KnowledgeScopeResp[]>(() => {
  const keyword = scopeKeyword.value.trim().toLowerCase()
  if (!keyword) {
    return scopes.value
  }
  return scopes.value.filter((item) => {
    const content = [item.scopeCode, item.scopeName, item.description, item.aliases].filter(Boolean).join(' ').toLowerCase()
    return content.includes(keyword)
  })
})
const filteredTopics = computed<KnowledgeTopicResp[]>(() => {
  const keyword = topicKeyword.value.trim().toLowerCase()
  if (!activeScopeCode.value) {
    return topics.value.filter((item) => {
      if (!keyword) {
        return true
      }
      const content = [item.topicCode, item.topicName, item.description, item.aliases].filter(Boolean).join(' ').toLowerCase()
      return content.includes(keyword)
    })
  }
  return topics.value.filter((item) => {
    if (item.scopeCode !== activeScopeCode.value) {
      return false
    }
    if (!keyword) {
      return true
    }
    const content = [item.topicCode, item.topicName, item.description, item.aliases].filter(Boolean).join(' ').toLowerCase()
    return content.includes(keyword)
  })
})
const filteredDocuments = computed<DocumentDetailResp[]>(() => {
  const keyword = documentKeyword.value.trim().toLowerCase()
  return documents.value.filter((item) => {
    if (activeScopeCode.value && item.knowledgeScopeCode !== activeScopeCode.value) {
      return false
    }
    if (!keyword) {
      return true
    }
    const content = [
      item.documentName,
      item.originalFileName,
      item.knowledgeScopeCode,
      item.knowledgeScopeName,
      item.businessCategory,
      item.documentTags
    ].filter(Boolean).join(' ').toLowerCase()
    return content.includes(keyword)
  })
})
const selectedProfileDocument = computed<DocumentDetailResp | null>(() => documents.value.find((item) => item.documentId === profileDocumentId.value) || null)
const selectedProfileDocumentMeta = computed<string>(() => {
  if (!selectedProfileDocument.value) {
    return '未选择文档'
  }
  return documentMetaLine(selectedProfileDocument.value)
})

const relations = computed<TopicDocumentRelationResp[]>(() => {
  const keyword = relationKeyword.value.trim().toLowerCase()
  return allRelations.value.filter((item) => {
    const topic = topics.value.find((topicItem) => topicItem.topicCode === item.topicCode)
    if (activeScopeCode.value && topic?.scopeCode !== activeScopeCode.value) {
      return false
    }
    if (activeTopicCode.value && item.topicCode !== activeTopicCode.value) {
      return false
    }
    if (!keyword) {
      return true
    }
    const content = [
      item.topicCode,
      item.documentName,
      item.reason,
      item.knowledgeScopeName,
      item.businessCategory,
      item.documentTags
    ].filter(Boolean).join(' ').toLowerCase()
    return content.includes(keyword)
  })
})

interface ScopeCoverage {
  scopeCode: string
  scopeName: string
  topicCount: number
  coveredTopicCount: number
  pendingTopicCount: number
  documentCount: number
  relationCount: number
  coverageRate: number
  coverageRateText: string
}

const scopeCoverageRows = computed<ScopeCoverage[]>(() => {
  return scopes.value.map((scope) => {
    const scopeTopics = topics.value.filter((topic) => topic.scopeCode === scope.scopeCode)
    const topicCodes = new Set(scopeTopics.map((topic) => topic.topicCode))
    const scopeRelations = allRelations.value.filter((relation) => topicCodes.has(relation.topicCode))
    const coveredTopicCodes = new Set(scopeRelations.map((relation) => relation.topicCode))
    const scopeDocuments = documents.value.filter((document) => document.knowledgeScopeCode === scope.scopeCode)
    const coverageRate = scopeTopics.length ? (coveredTopicCodes.size / scopeTopics.length) * 100 : 0
    return {
      scopeCode: scope.scopeCode,
      scopeName: scope.scopeName,
      topicCount: scopeTopics.length,
      coveredTopicCount: coveredTopicCodes.size,
      pendingTopicCount: Math.max(0, scopeTopics.length - coveredTopicCodes.size),
      documentCount: scopeDocuments.length,
      relationCount: scopeRelations.length,
      coverageRate,
      coverageRateText: `${coverageRate.toFixed(0)}%`
    }
  })
})
const overallCoverageRateText = computed<string>(() => {
  const totalTopics = topics.value.length
  if (!totalTopics) {
    return '0%'
  }
  const coveredTopicCodes = new Set(allRelations.value.map((relation) => relation.topicCode))
  return `${((coveredTopicCodes.size / totalTopics) * 100).toFixed(0)}%`
})

interface ProfileAnomaly {
  documentId: string
  documentName: string
  scopeText: string
  problems: string[]
  tone: 'danger' | 'warning'
  suggestion: string
}

const profileAnomalyRows = computed<ProfileAnomaly[]>(() => {
  const scopeCodes = new Set(scopes.value.map((scope) => scope.scopeCode))
  const linkedDocumentIds = new Set(allRelations.value.map((relation) => String(relation.documentId)))
  return documents.value
    .map((document) => {
      const problems: string[] = []
      if (!document.knowledgeScopeCode && !document.knowledgeScopeName) {
        problems.push('缺少知识范围')
      }
      if (document.knowledgeScopeCode && !scopeCodes.has(document.knowledgeScopeCode)) {
        problems.push('范围未建节点')
      }
      if (!document.businessCategory) {
        problems.push('缺少业务分类')
      }
      if (!document.documentTags) {
        problems.push('缺少标签')
      }
      if (!linkedDocumentIds.has(String(document.documentId))) {
        problems.push('未绑定主题')
      }
      const scopeText = document.knowledgeScopeName || document.knowledgeScopeCode || '未分配范围'
      const tone: 'danger' | 'warning' = problems.length > 0 ? 'danger' : 'warning'
      return {
        documentId: String(document.documentId),
        documentName: document.documentName,
        scopeText,
        problems,
        tone,
        suggestion: buildAnomalySuggestion(problems)
      }
    })
    .filter((item) => item.problems.length > 0)
})
const allVisibleAnomaliesSelected = computed<boolean>(() => {
  if (!profileAnomalyRows.value.length) {
    return false
  }
  return profileAnomalyRows.value.every((item) => selectedProfileRepairIds.value.includes(item.documentId))
})

interface SummaryCard {
  label: string
  value: string
  description: string
}

const summaryCards = computed<SummaryCard[]>(() => {
  const documentWithMetaCount = documents.value.filter((item) => {
    return Boolean(item.knowledgeScopeCode || item.knowledgeScopeName || item.businessCategory || item.documentTags)
  }).length
  const pendingTopicCount = topics.value.filter((item) => {
    return !allRelations.value.some((relation) => relation.topicCode === item.topicCode)
  }).length

  return [
    {
      label: '知识范围',
      value: String(scopes.value.length),
      description: '知识范围是自动路由的第一层收敛边界'
    },
    {
      label: '知识主题',
      value: String(topics.value.length),
      description: '主题是范围里的可回答单元'
    },
    {
      label: '文档数',
      value: String(documents.value.length),
      description: '当前可维护画像和路由元数据的文档数量'
    },
    {
      label: '已补元数据文档',
      value: String(documentWithMetaCount),
      description: '至少填了范围、业务类目或标签的文档数'
    },
    {
      label: '已保存关联',
      value: String(allRelations.value.length),
      description: '当前所有主题已保存的文档关联数'
    },
    {
      label: '未关联主题',
      value: String(pendingTopicCount),
      description: '还没有绑定任何文档关系的主题数'
    }
  ]
})

watch(
  () => relationForm.topicCode,
  (value: string) => {
    activeTopicCode.value = value || ''
    const currentTopic = topics.value.find((item) => item.topicCode === value)
    if (currentTopic?.scopeCode) {
      activeScopeCode.value = currentTopic.scopeCode
    }
  }
)

function showNotice(message: string, type: Notice['type'] = 'info'): void {
  notice.message = message
  notice.type = type
}

function resetScopeForm(): void {
  Object.assign(scopeForm, {
    scopeCode: '',
    scopeName: '',
    parentScopeCode: '',
    description: '',
    aliases: '',
    examples: '',
    sortOrder: 0,
    operatorId: OPERATOR_ID
  })
  activeScopeCode.value = ''
}

function resetTopicForm(): void {
  Object.assign(topicForm, {
    topicCode: '',
    topicName: '',
    scopeCode: activeScopeCode.value || '',
    description: '',
    aliases: '',
    examples: '',
    answerShape: '',
    executionPreference: '',
    sortOrder: 0,
    operatorId: OPERATOR_ID
  })
  activeTopicCode.value = ''
}

function editScope(item: KnowledgeScopeResp): void {
  activeScopeCode.value = item.scopeCode
  if (activeTopic.value && activeTopic.value.scopeCode !== item.scopeCode) {
    activeTopicCode.value = ''
    relationForm.topicCode = ''
  }
  Object.assign(scopeForm, {...item, operatorId: OPERATOR_ID})
  topicForm.scopeCode = item.scopeCode
}

function editTopic(item: KnowledgeTopicResp): void {
  activeScopeCode.value = item.scopeCode
  activeTopicCode.value = item.topicCode
  relationForm.topicCode = item.topicCode
  Object.assign(topicForm, {...item, operatorId: OPERATOR_ID})
}

function selectDocument(item: DocumentDetailResp): void {
  profileDocumentId.value = item.documentId
  profile.value = null
}

async function withAction<T>(task: () => Promise<T>, successMessage = ''): Promise<T | null> {
  actionLoading.value = true
  try {
    const result = await task()
    if (successMessage) {
      showNotice(successMessage, 'success')
    }
    return result
  } catch (error) {
    showNotice((error as Error).message || '执行失败', 'danger')
    return null
  } finally {
    actionLoading.value = false
  }
}

async function loadAll(): Promise<void> {
  loading.value = true
  try {
    const [scopeRes, topicRes, docRes] = await Promise.all([
      knowledgeApi.listScopes(),
      knowledgeApi.listTopics(),
      knowledgeApi.listTopicDocuments({topicCode: ''})
    ])
    scopes.value = scopeRes.data || []
    topics.value = topicRes.data || []
    allRelations.value = docRes.data || []
    await loadDocuments()

    if (activeScopeCode.value && !scopes.value.some((item) => item.scopeCode === activeScopeCode.value)) {
      activeScopeCode.value = ''
    }
    if (activeTopicCode.value && !topics.value.some((item) => item.topicCode === activeTopicCode.value)) {
      activeTopicCode.value = ''
      relationForm.topicCode = ''
    }
  } catch (error) {
    showNotice((error as Error).message || '加载知识路由数据失败', 'danger')
  } finally {
    loading.value = false
  }
}

async function loadDocuments(): Promise<void> {
  try {
    const res = await documentApi.queryDocumentPage({pageNo: 1, pageSize: 200})
    documents.value = res.data?.records || []
  } catch (error) {
    console.error('加载文档列表失败', error)
    documents.value = []
  }
}

async function saveScope(): Promise<void> {
  await withAction(async () => {
    const res = await knowledgeApi.saveScope(scopeForm)
    const data = res.data as KnowledgeScopeResp
    activeScopeCode.value = data?.scopeCode || scopeForm.scopeCode
    await loadAll()
    closeDrawer()
  }, '知识范围已保存')
}

async function deleteScope(): Promise<void> {
  if (!activeScope.value || !window.confirm(`确认删除范围「${activeScope.value.scopeName}」吗？`)) {
    return
  }
  await withAction(async () => {
    await knowledgeApi.deleteScope({
      scopeCode: activeScope.value?.scopeCode || '',
      operatorId: OPERATOR_ID
    })
    resetScopeForm()
    closeDrawer()
    await loadAll()
  }, '知识范围已删除')
}

async function saveTopic(): Promise<void> {
  await withAction(async () => {
    const res = await knowledgeApi.saveTopic(topicForm)
    const data = res.data as KnowledgeTopicResp
    activeTopicCode.value = data?.topicCode || topicForm.topicCode
    relationForm.topicCode = activeTopicCode.value
    await loadAll()
    closeDrawer()
  }, '知识主题已保存')
}

async function deleteTopic(): Promise<void> {
  if (!activeTopic.value || !window.confirm(`确认删除主题「${activeTopic.value.topicName}」吗？`)) {
    return
  }
  await withAction(async () => {
    await knowledgeApi.deleteTopic({
      topicCode: activeTopic.value?.topicCode || '',
      operatorId: OPERATOR_ID
    })
    resetTopicForm()
    relationForm.topicCode = ''
    closeDrawer()
    await loadAll()
  }, '知识主题已删除')
}

async function loadProfile(): Promise<void> {
  if (!profileDocumentId.value) {
    return
  }
  await withAction(async () => {
    const {data} = await documentApi.getProfile({documentId: profileDocumentId.value})
    profile.value = data || null
  })
}

async function regenerateProfile(): Promise<void> {
  if (!profileDocumentId.value) {
    return
  }
  await withAction(async () => {
    const res = await documentApi.regenerateProfile({
      documentId: profileDocumentId.value,
      operatorId: OPERATOR_ID
    })
    profile.value = res.data || null
  }, '文档画像已重新生成')
}

async function regenerateAllProfiles(): Promise<void> {
  if (!documents.value.length || !window.confirm(`确认批量重建 ${documents.value.length} 份文档画像吗？`)) {
    return
  }
  batchLoading.value = true
  try {
    await documentApi.batchRegenerateProfile({
      documentIds: documents.value.map((item) => item.documentId),
      operatorId: OPERATOR_ID
    })
    showNotice(`已触发 ${documents.value.length} 份文档的画像重建`, 'success')
    if (profileDocumentId.value) {
      await loadProfile()
    }
  } catch (error) {
    showNotice((error as Error).message || '批量重建文档画像失败', 'danger')
  } finally {
    batchLoading.value = false
  }
}

async function batchRepairProfiles(): Promise<void> {
  const documentIds = [...selectedProfileRepairIds.value]
  if (!documentIds.length) {
    showNotice('请先选择要批量修复的文档。', 'danger')
    return
  }
  batchLoading.value = true
  try {
    await documentApi.batchRegenerateProfile({
      documentIds,
      operatorId: OPERATOR_ID
    })
    selectedProfileRepairIds.value = []
    showNotice(`已批量重建 ${documentIds.length} 份文档画像。`, 'success')
    if (profileDocumentId.value) {
      await loadProfile()
    }
    await loadAll()
  } catch (error) {
    showNotice((error as Error).message || '批量重建文档画像失败', 'danger')
  } finally {
    batchLoading.value = false
  }
}

async function loadRelations(): Promise<void> {
  try {
    const res = await knowledgeApi.listTopicDocuments({topicCode: ''})
    allRelations.value = res.data || []
  } catch (error) {
    showNotice((error as Error).message || '加载主题文档关联失败', 'danger')
  }
}

async function saveRelation(): Promise<void> {
  await withAction(async () => {
    await knowledgeApi.saveTopicDocument(relationForm)
    await loadRelations()
    closeDrawer()
  }, '主题文档关联已保存')
}

async function removeRelation(item: TopicDocumentRelationResp | null): Promise<void> {
  await withAction(async () => {
    await knowledgeApi.removeTopicDocument({
      topicCode: item?.topicCode || '',
      documentId: item?.documentId || '',
      operatorId: OPERATOR_ID
    })
    await loadRelations()
  }, '主题文档关联已移除')
}

function documentMetaLine(item: Partial<DocumentDetailResp> = {}): string {
  return [item.knowledgeScopeName || item.knowledgeScopeCode, item.businessCategory, item.documentTags]
    .filter(Boolean)
    .join(' · ') || '还没有范围 / 类目 / 标签元数据'
}

function parseTextList(value: unknown): string[] {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return []
  }
  return normalized
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatAnswerShapeLabel(value: string): string {
  return formatMappedLabel(value, ANSWER_SHAPE_LABEL_MAP)
}

function formatExecutionPreferenceLabel(value: string): string {
  return formatMappedLabel(value, EXECUTION_PREFERENCE_LABEL_MAP)
}

function formatDocumentTypeLabel(value: string): string {
  return formatMappedLabel(value, DOCUMENT_TYPE_LABEL_MAP)
}

function formatProfileSourceLabel(value: string): string {
  return formatMappedLabel(value, PROFILE_SOURCE_LABEL_MAP)
}

function formatMappedLabel(value: string, labelMap: Record<string, string>): string {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return '未设置'
  }
  return labelMap[normalized] || normalized
}

function buildAnomalySuggestion(problems: string[]): string {
  if (problems.includes('范围未建节点')) {
    return '建议先在知识范围区补齐对应 scopeCode，再重建画像并复测自动路由。'
  }
  if (problems.includes('缺少知识范围') || problems.includes('缺少标签')) {
    return '建议重新上传时补齐知识范围和文档标签；当前可先重建画像观察自动补全效果。'
  }
  if (problems.includes('未绑定主题')) {
    return '建议在主题文档关联区为该文档至少绑定 1 个核心主题。'
  }
  return '建议重建画像后查看核心主题、示例问题和图能力是否恢复正常。'
}

function toggleProfileRepair(documentId: string | number): void {
  const normalized = String(documentId)
  if (selectedProfileRepairIds.value.includes(normalized)) {
    selectedProfileRepairIds.value = selectedProfileRepairIds.value.filter((item) => item !== normalized)
    return
  }
  selectedProfileRepairIds.value = [...selectedProfileRepairIds.value, normalized]
}

function toggleAllVisibleAnomalies(): void {
  if (allVisibleAnomaliesSelected.value) {
    const visibleIds = new Set(profileAnomalyRows.value.map((item) => item.documentId))
    selectedProfileRepairIds.value = selectedProfileRepairIds.value.filter((item) => !visibleIds.has(item))
    return
  }
  const merged = new Set(selectedProfileRepairIds.value)
  profileAnomalyRows.value.forEach((item) => merged.add(item.documentId))
  selectedProfileRepairIds.value = [...merged]
}

function selectAnomalyDocument(item: ProfileAnomaly): void {
  profileDocumentId.value = item.documentId
  profile.value = null
  loadProfile()
}

function parseJsonArray(value: string): string[] {
  if (!value) {
    return []
  }
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed.filter(Boolean) : []
  } catch {
    return []
  }
}

function graphCapabilityText(profileValue: DocumentProfileResp): string {
  const enabled: string[] = []
  if (profileValue.supportsGraphOutline) {
    enabled.push('大纲导航')
  }
  if (profileValue.supportsItemLookup) {
    enabled.push('条目定位')
  }
  if (profileValue.supportsGraphAssist) {
    enabled.push('图辅助检索')
  }
  return enabled.length ? enabled.join(' / ') : '未开启'
}

function profileStatusText(status: number): string {
  if (status === 2) {
    return '已生成'
  }
  if (status === 3) {
    return '生成失败'
  }
  return '待生成'
}

function profileStatusClass(status: number): string {
  if (status === 2) {
    return 'profile-status-success'
  }
  if (status === 3) {
    return 'profile-status-danger'
  }
  return 'profile-status-warning'
}

onMounted(loadAll)
</script>

<style scoped>
.knowledge-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header,
.panel-card,
.stat-card {
  background: #fff;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 12px);
  box-shadow: var(--shadow-sm);
}

.page-header {
  padding: 24px 26px;
  display: flex;
  justify-content: space-between;
  gap: 20px;
}

.page-header h3,
.panel-card h4 {
  margin: 0;
  color: var(--color-text-strong);
}

.page-header p,
.panel-head p,
.stat-card small,
.empty-inline,
.document-row span,
.document-row small,
.list-row span,
.list-row small,
.relation-row span,
.relation-row small,
.profile-summary,
.summary-label {
  color: var(--color-muted);
}

.header-actions,
.actions,
.helper-bar,
.tag-list {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.workbench-nav {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.workbench-pill {
  border: 1px solid rgba(37, 87, 214, 0.12);
  border-radius: 999px;
  padding: 9px 14px;
  background: rgba(37, 87, 214, 0.06);
  color: var(--color-primary-strong);
  font-weight: 600;
  cursor: pointer;
}

.coverage-panel,
.anomaly-panel {
  background: #fff;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 12px);
  box-shadow: var(--shadow-sm);
  padding: 18px;
}

.section-card-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.section-card-head h4 {
  margin: 6px 0 0;
  color: var(--color-text-strong);
}

.section-card-head p {
  margin: 8px 0 0;
  color: var(--color-muted);
  line-height: 1.7;
}

.coverage-grid,
.anomaly-list {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.coverage-card,
.anomaly-card {
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.88);
  padding: 14px;
}

.coverage-card.warning,
.anomaly-card.warning {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.07), rgba(255, 255, 255, 0.9));
}

.anomaly-card.danger {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.06), rgba(255, 255, 255, 0.92));
}

.coverage-head,
.anomaly-card {
  display: grid;
  gap: 10px;
}

.coverage-head strong,
.coverage-rate,
.coverage-metrics span,
.anomaly-main strong {
  color: var(--color-text-strong);
}

.coverage-head span,
.anomaly-main span,
.anomaly-main small {
  color: var(--color-muted);
  font-size: 12px;
}

.coverage-track {
  margin-top: 10px;
  height: 8px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  overflow: hidden;
}

.coverage-track span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #2557d6, #0f766e);
}

.coverage-rate {
  font-weight: 700;
}

.coverage-metrics,
.batch-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.anomaly-card {
  grid-template-columns: auto 1fr auto;
  align-items: flex-start;
  gap: 12px;
}

.anomaly-main {
  display: grid;
  gap: 8px;
}

.row-check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.row-check input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.row-check span {
  width: 18px;
  height: 18px;
  border-radius: 6px;
  border: 1px solid rgba(37, 87, 214, 0.22);
  background: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.row-check input:checked + span {
  background: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: inset 0 0 0 4px #fff;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}

.stat-card {
  padding: 16px;
  display: grid;
  gap: 8px;
}

.stat-card span,
.profile-head span,
.mini-card span,
.tag-section p {
  font-size: 12px;
  color: var(--color-muted);
}

.stat-card strong {
  color: var(--color-text-strong);
  font-size: 22px;
}

.page-notice {
  padding: 10px 12px;
  border-radius: var(--radius-sm, 8px);
}

.page-notice-success {
  background: #ecfdf3;
  color: #027a48;
}

.page-notice-danger {
  background: #fef3f2;
  color: #b42318;
}

.grid {
  display: grid;
  gap: 16px;
}

.grid.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.panel-card {
  padding: 22px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.helper-bar {
  margin-top: 14px;
}

.toolbar-row {
  margin-top: 14px;
  display: grid;
  gap: 10px;
}

.toolbar-row input {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 8px);
  padding: 10px 12px;
  color: var(--color-text-strong);
  background: #fff;
}

.toolbar-row-triple {
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
}

.relation-toolbar {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}

.helper-pill {
  display: inline-flex;
  align-items: center;
  padding: 7px 12px;
  border-radius: 999px;
  background: rgba(37, 87, 214, 0.08);
  color: var(--color-primary-strong);
  font-size: 12px;
  font-weight: 600;
}

.helper-pill-soft {
  background: var(--color-surface-soft);
  color: var(--color-text);
}

.form-grid {
  margin-top: 16px;
  display: grid;
  gap: 10px;
}

.form-grid input,
.form-grid select,
.form-grid textarea {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 8px);
  padding: 10px 12px;
  background: #fff;
  color: var(--color-text-strong);
}

.form-grid textarea {
  min-height: 74px;
  resize: vertical;
}

.insight-card,
.profile-health-card {
  margin-top: 16px;
  padding: 16px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: linear-gradient(135deg, rgba(37, 87, 214, 0.05), rgba(239, 123, 57, 0.05));
  display: grid;
  gap: 14px;
}

.insight-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
}

.insight-head strong,
.insight-description {
  color: var(--color-text-strong);
}

.insight-kicker {
  display: block;
  color: var(--color-muted);
  font-size: 12px;
  margin-bottom: 4px;
}

.insight-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.mini-stat {
  padding: 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.74);
  border: 1px solid rgba(17, 24, 39, 0.08);
  display: grid;
  gap: 6px;
}

.mini-stat span {
  color: var(--color-muted);
  font-size: 12px;
}

.mini-stat strong {
  color: var(--color-text-strong);
}

.simple-list,
.document-list {
  margin-top: 16px;
  display: grid;
  gap: 8px;
  max-height: 360px;
  overflow: auto;
}

.list-row,
.document-row {
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-surface-soft);
  padding: 12px;
  text-align: left;
  cursor: pointer;
  display: grid;
  gap: 6px;
}

.list-row.active,
.document-row.active {
  border-color: rgba(37, 87, 214, 0.18);
  box-shadow: inset 0 0 0 1px rgba(37, 87, 214, 0.08);
  background: rgba(37, 87, 214, 0.04);
}

.topic-meta-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.topic-meta-row span {
  color: var(--color-muted);
  font-size: 12px;
}

.list-row strong,
.document-row strong,
.relation-row strong,
.profile-head strong,
.mini-card strong {
  color: var(--color-text-strong);
}

.profile-card,
.mini-card,
.relation-row {
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-surface-soft);
}

.profile-card {
  margin-top: 16px;
  padding: 16px;
  display: grid;
  gap: 16px;
}

.profile-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.profile-status-pill {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.profile-status-success {
  background: rgba(34, 197, 94, 0.12);
  color: #15803d;
}

.profile-status-warning {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.profile-status-danger {
  background: rgba(239, 68, 68, 0.12);
  color: #b91c1c;
}

.profile-summary {
  margin: 0;
  line-height: 1.75;
}

.profile-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.mini-card {
  padding: 12px;
  display: grid;
  gap: 6px;
}

.tag-section {
  display: grid;
  gap: 8px;
}

.tag-section p {
  margin: 0;
}

.tag-chip {
  display: inline-flex;
  align-items: center;
  padding: 7px 10px;
  border-radius: 999px;
  background: rgba(37, 87, 214, 0.08);
  color: var(--color-primary-strong);
  font-size: 12px;
}

.tag-chip-soft {
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-text);
}

.tag-chip-empty {
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-muted);
}

.tag-chip-warning {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.relation-list {
  max-height: 420px;
}

.relation-row {
  padding: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.primary-button,
.ghost-button,
.danger-link {
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 10px 16px;
  font-weight: 600;
  cursor: pointer;
}

.primary-button {
  background: var(--color-primary);
  color: #fff;
}

.ghost-button {
  background: #fff;
  color: var(--color-text);
  border-color: var(--color-border);
}

.toggle-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 999px;
  background: var(--color-surface-soft);
  color: var(--color-text);
  font-size: 13px;
}

.toggle-chip input {
  margin: 0;
}

.ghost-danger,
.danger-link {
  color: var(--color-danger);
  border-color: rgba(179, 76, 47, 0.14);
  background: rgba(179, 76, 47, 0.06);
}

.danger-link {
  padding: 8px 14px;
}

.primary-button:disabled,
.ghost-button:disabled,
.danger-link:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 1080px) {

  .grid.two,
  .profile-grid,
  .insight-stats,
  .toolbar-row-triple,
  .relation-toolbar {
    grid-template-columns: 1fr;
  }

  .coverage-grid,
  .anomaly-list {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {

  .page-header,
  .panel-head,
  .profile-head,
  .section-card-head {
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
  }

  .relation-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .anomaly-card {
    grid-template-columns: 1fr;
  }
}

/* Tab Navigation */
.tab-nav {
  display: flex;
  gap: 4px;
  background: #fff;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 12px);
  padding: 6px;
  box-shadow: var(--shadow-sm);
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
}

.tab-btn:hover {
  background: rgba(37, 87, 214, 0.04);
}

.tab-btn.active {
  background: rgba(37, 87, 214, 0.08);
}

.tab-step {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: rgba(15, 23, 42, 0.08);
  color: var(--color-muted);
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.tab-btn.active .tab-step {
  background: var(--color-primary);
  color: #fff;
}

.tab-label {
  font-weight: 600;
  color: var(--color-text-strong);
  white-space: nowrap;
}

.tab-hint {
  font-size: 12px;
  color: var(--color-muted);
  white-space: nowrap;
}

/* Tab Content */
.tab-content {
  animation: tabFadeIn 0.2s ease;
}

@keyframes tabFadeIn {
  from {
    opacity: 0;
    transform: translateY(4px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Card Grid */
.card-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
  max-height: 520px;
  overflow-y: auto;
}

.data-card {
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-surface-soft);
  padding: 14px;
  cursor: pointer;
  display: grid;
  gap: 8px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.data-card:hover {
  border-color: rgba(37, 87, 214, 0.2);
  box-shadow: 0 2px 8px rgba(37, 87, 214, 0.06);
}

.data-card.active {
  border-color: rgba(37, 87, 214, 0.3);
  background: rgba(37, 87, 214, 0.04);
}

.data-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.data-card-head strong {
  color: var(--color-text-strong);
}

.data-card-meta {
  display: flex;
  gap: 12px;
}

.data-card-meta span {
  font-size: 12px;
  color: var(--color-muted);
}

.data-card small {
  color: var(--color-muted);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* Relation Table */
.relation-table {
  margin-top: 16px;
  display: grid;
  gap: 8px;
  max-height: 520px;
  overflow-y: auto;
}

/* Filter toolbar */
.toolbar-row-filters {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.filter-select {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 8px);
  padding: 10px 12px;
  background: #fff;
  color: var(--color-text-strong);
}

/* Collapse arrow */
.coverage-toggle-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.collapse-arrow {
  display: inline-block;
  font-size: 12px;
  color: var(--color-muted);
  transition: transform 0.2s;
}

.collapse-arrow.collapsed {
  transform: rotate(-90deg);
}

/* Drawer */
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.3);
  z-index: 50;
}

.drawer-panel {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: 480px;
  max-width: 90vw;
  background: #fff;
  box-shadow: -4px 0 24px rgba(15, 23, 42, 0.12);
  z-index: 51;
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border);
}

.drawer-header h4 {
  margin: 0;
  color: var(--color-text-strong);
}

.drawer-close {
  padding: 6px 12px;
  font-size: 13px;
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
}

.drawer-footer {
  display: flex;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--color-border);
}

.drawer-detail {
  display: grid;
  gap: 14px;
}

.detail-row {
  display: grid;
  gap: 4px;
}

.detail-row span {
  font-size: 12px;
  color: var(--color-muted);
}

.detail-row strong {
  color: var(--color-text-strong);
}

.detail-row p {
  margin: 0;
  color: var(--color-text-strong);
  line-height: 1.6;
}

/* Drawer transitions */
.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 0.25s ease;
}

.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}

.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 0.25s ease;
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 1080px) {
  .tab-nav {
    flex-wrap: wrap;
  }

  .tab-hint {
    display: none;
  }

  .toolbar-row-filters {
    grid-template-columns: 1fr;
  }

  .card-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .tab-btn {
    padding: 10px 12px;
  }

  .tab-label {
    font-size: 13px;
  }

  .drawer-panel {
    width: 100vw;
    max-width: 100vw;
  }
}
</style>
