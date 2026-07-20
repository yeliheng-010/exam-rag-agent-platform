<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/classes')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回班级
        </t-button>
        <h2>{{ classInfo?.name || '班级详情' }}</h2>
        <p>{{ classInfo?.description || '围绕班级空间组织资料、题库、作业与学习分析。' }}</p>
      </div>
      <t-tag v-if="classInfo" :theme="classInfo.status === 'active' ? 'success' : 'default'" variant="light">
        {{ classInfo.status === 'active' ? '运行中' : '已归档' }}
      </t-tag>
    </div>

    <t-loading :loading="loading">
      <div v-if="classInfo" class="summary-grid">
        <div class="summary-item">
          <span>班级空间</span>
          <strong>{{ classInfo.space_id }}</strong>
        </div>
        <div class="summary-item">
          <span>成员上限</span>
          <strong>{{ classInfo.member_limit || '不限' }}</strong>
        </div>
        <div class="summary-item">
          <span>邀请码</span>
          <strong>{{ classInfo.invite_code || '待生成' }}</strong>
        </div>
      </div>

      <t-tabs v-model="activeTab" class="detail-tabs">
        <t-tab-panel value="overview" label="概览">
          <div class="tab-panel">
            <h3>第一阶段班级承载能力</h3>
            <p>班级已拥有独立 exam_space，后续知识库、题库、作业、权益都会通过 space_id 进行隔离。</p>
            <div class="flow-grid">
              <div>班级空间</div>
              <div>资料入库</div>
              <div>试卷结构化</div>
              <div>作业与练习</div>
            </div>
          </div>
        </t-tab-panel>
        <t-tab-panel value="members" label="成员" :disabled="isArchivedClass">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>成员审核</h3>
                <p>学生通过邀请码提交加入申请，老师或助教审核通过后才能进入班级。</p>
              </div>
              <t-button v-if="canReviewMembers" variant="outline" :loading="membersLoading" @click="loadMembers">
                <template #icon><t-icon name="refresh" /></template>
                刷新
              </t-button>
            </div>
            <t-alert
              v-if="!canReviewMembers"
              theme="info"
              message="当前账号可查看班级信息，成员审核由班级老师或助教处理。"
            />
            <t-loading v-else :loading="membersLoading">
              <t-table
                row-key="id"
                :data="members"
                :columns="memberColumns"
                :pagination="{ pageSize: 8, total: members.length }"
                size="small"
              >
                <template #member="{ row }">
                  <div class="resource-name-cell">
                    <strong>{{ memberDisplayName(row) }}</strong>
                    <span>{{ memberDisplayId(row) }}</span>
                  </div>
                </template>
                <template #role="{ row }">
                  <t-tag variant="light" :theme="roleTheme(row.role)">{{ roleLabel(row.role) }}</t-tag>
                </template>
                <template #status="{ row }">
                  <t-tag variant="light" :theme="row.status === 'pending' ? 'warning' : 'success'">
                    {{ row.status === 'pending' ? '待审核' : '已加入' }}
                  </t-tag>
                </template>
                <template #created_at="{ row }">
                  {{ formatDate(row.created_at) }}
                </template>
                <template #actions="{ row }">
                  <t-space v-if="row.status === 'pending'" size="small">
                    <t-button size="small" theme="primary" :loading="reviewingUserId === row.user_id" @click="approveMember(row.user_id)">通过</t-button>
                    <t-button size="small" theme="danger" variant="outline" :loading="reviewingUserId === row.user_id" @click="rejectMember(row.user_id)">拒绝</t-button>
                  </t-space>
                  <span v-else class="muted-text">无操作</span>
                </template>
              </t-table>
            </t-loading>
          </div>
        </t-tab-panel>
        <t-tab-panel value="resources" label="资料" :disabled="isArchivedClass">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>班级知识库</h3>
                <p>老师将知识库绑定到班级空间后，审核通过的学生可以在学习中心选择这些资料进行基础 RAG 对话。</p>
              </div>
              <t-space size="small">
                <t-button variant="outline" :loading="resourcesLoading || materialsLoading" @click="loadResourceTab">
                  <template #icon><t-icon name="refresh" /></template>
                  刷新
                </t-button>
                <t-button v-if="canManageResources" theme="primary" @click="openBindDialog">
                  <template #icon><t-icon name="link" /></template>
                  绑定知识库
                </t-button>
                <t-button v-if="canManageResources" theme="primary" variant="outline" @click="openMaterialDialog">
                  <template #icon><t-icon name="file-add" /></template>
                  登记资料
                </t-button>
              </t-space>
            </div>
            <t-alert
              v-if="!canManageResources"
              theme="info"
              message="当前账号可使用班级资料进行对话，资料配置由班级老师或助教处理。"
            />
            <t-loading :loading="resourcesLoading">
              <t-table
                row-key="id"
                :data="resources"
                :columns="resourceColumns"
                :pagination="{ pageSize: 8, total: resources.length }"
                size="small"
              >
                <template #resource_id="{ row }">
                  <div class="resource-name-cell">
                    <strong>{{ knowledgeBaseName(row.resource_id) }}</strong>
                    <span>{{ row.resource_id }}</span>
                  </div>
                </template>
                <template #material_type="{ row }">
                  <t-tag variant="light">{{ materialTypeLabel(row.material_type) }}</t-tag>
                </template>
                <template #domain_id="{ row }">
                  {{ domainName(row.domain_id) }}
                </template>
                <template #review_status="{ row }">
                  <t-tag :theme="reviewTag(row.review_status)" variant="light">{{ reviewLabel(row.review_status) }}</t-tag>
                </template>
                <template #updated_at="{ row }">
                  {{ formatDate(row.updated_at) }}
                </template>
                <template #actions="{ row }">
                  <t-button variant="text" size="small" @click="startChat(row.resource_id)">开始对话</t-button>
                </template>
              </t-table>
              <t-empty v-if="!resources.length && !resourcesLoading" size="small" description="暂无班级资料" />
            </t-loading>

            <t-divider />

            <div class="resource-section">
              <div class="section-title-row">
                <div>
                  <h3>考试资料</h3>
                  <p>具体到某份试卷、答案或学习资料的登记记录，用于后续结构化、题库生成和学习分析。</p>
                </div>
              </div>
              <t-loading :loading="materialsLoading">
                <t-table
                  row-key="id"
                  :data="materials"
                  :columns="materialColumns"
                  :pagination="{ pageSize: 8, total: materials.length }"
                  size="small"
                >
                  <template #title="{ row }">
                    <div class="resource-name-cell">
                      <strong>{{ row.title }}</strong>
                      <span>{{ knowledgeBaseName(row.knowledge_base_id) }} / {{ row.knowledge_id }}</span>
                    </div>
                  </template>
                  <template #material_type="{ row }">
                    <t-tag variant="light">{{ materialTypeLabel(row.material_type) }}</t-tag>
                  </template>
                  <template #ingest_status="{ row }">
                    <t-tag :theme="ingestStatusTheme(row.ingest_status)" variant="light">
                      {{ ingestStatusLabel(row.ingest_status) }}
                    </t-tag>
                  </template>
                  <template #domain_id="{ row }">
                    {{ domainName(row.domain_id) }}
                  </template>
                  <template #source_year="{ row }">
                    {{ row.source_year || '-' }}
                  </template>
                  <template #updated_at="{ row }">
                    {{ formatDate(row.updated_at) }}
                  </template>
                </t-table>
                <t-empty v-if="!materials.length && !materialsLoading" size="small" description="暂无考试资料" />
              </t-loading>
            </div>

            <div class="resource-section">
              <div class="section-title-row">
                <div>
                  <h3>结构化任务</h3>
                  <p>当前阶段先记录任务和可校对状态，后续再接入 LLM 抽题、答案匹配和人工审核工作台。</p>
                </div>
              </div>
              <t-loading :loading="materialsLoading">
                <t-table
                  row-key="id"
                  :data="structuringTasks"
                  :columns="taskColumns"
                  :pagination="{ pageSize: 8, total: structuringTasks.length }"
                  size="small"
                >
                  <template #material_id="{ row }">
                    <div class="resource-name-cell">
                      <strong>{{ taskMaterialTitle(row.material_id) }}</strong>
                      <span>{{ row.question_bank_id }}</span>
                    </div>
                  </template>
                  <template #status="{ row }">
                    <t-tag :theme="taskStatusTheme(row.status)" variant="light">
                      {{ taskStatusLabel(row.status) }}
                    </t-tag>
                    <div v-if="row.error_message" class="task-error">{{ row.error_message }}</div>
                    <div v-else-if="row.status === 'extracting'" class="task-error">
                      {{ row.progress?.message || '后台抽取中' }} · {{ row.progress?.percent || 0 }}%
                    </div>
                  </template>
                  <template #created_at="{ row }">
                    {{ formatDate(row.created_at) }}
                  </template>
                  <template #actions="{ row }">
                    <t-space size="small">
                      <t-button
                        v-if="canManageResources && ['ready_for_review', 'failed'].includes(row.status)"
                        size="small"
                        theme="primary"
                        :loading="extractingTaskId === row.id"
                        @click="extractTask(row)"
                      >
                        {{ row.status === 'failed' ? '重新抽取题组' : '开始题组抽取' }}
                      </t-button>
                      <t-button
                        v-if="canManageResources && ['extracting', 'reviewing', 'completed'].includes(row.status)"
                        size="small"
                        variant="outline"
                        @click="router.push(`/platform/question-group-drafts/${row.id}`)"
                      >
                        题组校对
                      </t-button>
                    </t-space>
                  </template>
                </t-table>
                <t-empty v-if="!structuringTasks.length && !materialsLoading" size="small" description="暂无结构化任务" />
              </t-loading>
            </div>
          </div>
        </t-tab-panel>
        <t-tab-panel value="assignments" label="练习任务" :disabled="isArchivedClass">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>班级练习任务</h3>
                <p>老师可把已确认的正式题组发布为练习任务，学生从任务入口进入后会记录本次作业关联。</p>
              </div>
              <t-space size="small">
                <t-button variant="outline" :loading="assignmentsLoading" @click="loadAssignments">
                  <template #icon><t-icon name="refresh" /></template>
                  刷新
                </t-button>
                <t-button v-if="canManageAssignments" theme="primary" @click="openAssignmentDialog">
                  <template #icon><t-icon name="add" /></template>
                  发布任务
                </t-button>
              </t-space>
            </div>
            <t-alert
              v-if="!canManageAssignments"
              theme="info"
              message="当前账号可查看并完成班级练习任务，任务发布由班级老师或助教处理。"
            />
            <t-loading :loading="assignmentsLoading">
              <div v-if="assignments.length" class="assignment-list">
                <div v-for="item in assignments" :key="item.assignment.id" class="assignment-card">
                  <div class="assignment-card__header">
                    <div class="assignment-card__title">
                      <strong>{{ item.assignment.title || assignmentGroupLabel(item) }}</strong>
                      <span>{{ item.bank_name || '题库' }} · {{ assignmentGroupLabel(item) }}</span>
                    </div>
                    <t-space size="small">
                      <t-tag variant="light" :theme="assignmentStatusTheme(item.assignment)">
                        {{ assignmentStatusLabel(item.assignment) }}
                      </t-tag>
                      <t-tag v-if="item.last_attempt" variant="light" :theme="item.last_attempt.status === 'completed' ? 'success' : 'warning'">
                        {{ assignmentProgress(item) }}
                      </t-tag>
                    </t-space>
                  </div>
                  <p v-if="item.assignment.instructions" class="assignment-card__instructions">
                    {{ item.assignment.instructions }}
                  </p>
                  <div class="assignment-card__footer">
                    <span>{{ item.question_count }} 题 · {{ assignmentDueText(item.assignment.due_at) }}</span>
                    <t-space v-if="canManageAssignments" class="assignment-card__actions" size="small">
                      <t-button
                        v-if="canEditAssignment(item.assignment)"
                        size="small"
                        variant="text"
                        @click="openEditAssignmentDialog(item)"
                      >
                        编辑
                      </t-button>
                      <t-button
                        size="small"
                        variant="outline"
                        :loading="loadingAssignmentProgressId === item.assignment.id"
                        @click="openAssignmentProgress(item)"
                      >
                        查看结果
                      </t-button>
                      <t-button
                        v-if="canWithdrawAssignment(item.assignment)"
                        size="small"
                        theme="danger"
                        variant="text"
                        :loading="assignmentActionId === item.assignment.id"
                        @click="confirmWithdrawAssignment(item)"
                      >
                        撤回
                      </t-button>
                      <t-button
                        v-if="item.assignment.status === 'withdrawn'"
                        size="small"
                        theme="primary"
                        :disabled="!canRepublishAssignment(item.assignment)"
                        :loading="assignmentActionId === item.assignment.id"
                        @click="submitRepublishAssignment(item)"
                      >
                        重新发布
                      </t-button>
                    </t-space>
                    <t-button
                      v-else
                      size="small"
                      theme="primary"
                      :disabled="!existingAssignmentAttemptID(item) && !canCreateAssignmentAttempt(item)"
                      :loading="startingAssignmentId === item.assignment.id"
                      @click="startAssignmentPractice(item)"
                    >
                      {{ assignmentStartLabel(item) }}
                    </t-button>
                  </div>
                </div>
              </div>
              <t-empty v-else-if="!assignmentsLoading" size="small" description="暂无班级练习任务" />
            </t-loading>
          </div>
        </t-tab-panel>
        <t-tab-panel value="analytics" label="分析" :disabled="isArchivedClass">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>班级练习分析</h3>
                <p>基于班级已发布练习任务和学生最新作答记录，汇总完成率、正确率与学生参与情况。</p>
              </div>
              <div v-if="canViewAnalytics" class="panel-actions">
                <ClassPracticeRecommendations :class-id="currentClassId" @publish="prefillRecommendedAssignment" />
                <t-button variant="outline" :loading="analyticsLoading" @click="loadClassAnalytics">
                  <template #icon><t-icon name="refresh" /></template>
                  刷新
                </t-button>
              </div>
            </div>
            <t-alert
              v-if="!canViewAnalytics"
              theme="info"
              message="当前账号可完成练习并查看自己的学习记录，全班分析由班级老师或助教查看。"
            />
            <t-loading v-else :loading="analyticsLoading">
              <div v-if="classAnalytics" class="analytics-panel">
                <div class="analytics-summary-grid">
                  <div class="summary-item">
                    <span>学生数</span>
                    <strong>{{ classAnalytics.total_students }}</strong>
                  </div>
                  <div class="summary-item">
                    <span>练习任务</span>
                    <strong>{{ classAnalytics.assignment_count }}</strong>
                  </div>
                  <div class="summary-item">
                    <span>完成率</span>
                    <strong>{{ formatPercent(classAnalytics.completion_rate) }}</strong>
                  </div>
                  <div class="summary-item">
                    <span>平均正确率</span>
                    <strong>{{ formatPercent(classAnalytics.average_correct_rate) }}</strong>
                  </div>
                  <div class="summary-item">
                    <span>进度槽位</span>
                    <strong>{{ classAnalytics.completed_count }}/{{ classAnalytics.total_assignment_slots }}</strong>
                  </div>
                </div>

                <div class="analytics-section">
                  <div class="section-title-row">
                    <div>
                      <h3>学生维度</h3>
                      <p>只统计已加入班级的学生，不纳入老师、助教或待审核成员。</p>
                    </div>
                  </div>
                  <t-table
                    row-key="student_key"
                    :data="classAnalyticsMembers"
                    :columns="analyticsMemberColumns"
                    :pagination="{ pageSize: 8, total: classAnalyticsMembers.length }"
                    size="small"
                  >
                    <template #student="{ row }">
                      <div class="resource-name-cell">
                        <strong>{{ memberDisplayName(row.member) }}</strong>
                        <span>{{ memberDisplayId(row.member) }} · {{ roleLabel(row.member.role) }}</span>
                      </div>
                    </template>
                    <template #progress="{ row }">
                      {{ row.completed_count }}/{{ row.assignment_count }}
                    </template>
                    <template #completion_rate="{ row }">
                      {{ formatPercent(row.completion_rate) }}
                    </template>
                    <template #average_correct_rate="{ row }">
                      {{ formatPercent(row.average_correct_rate) }}
                    </template>
                    <template #last_activity_at="{ row }">
                      {{ row.last_activity_at ? formatDate(row.last_activity_at) : '-' }}
                    </template>
                  </t-table>
                  <t-empty v-if="!classAnalyticsMembers.length && !analyticsLoading" size="small" description="暂无学生练习数据" />
                </div>

                <div class="analytics-section">
                  <div class="section-title-row">
                    <div>
                      <h3>任务维度</h3>
                      <p>按练习任务聚合学生开始数、完成人数和平均正确率，用于快速定位需要讲评的任务。</p>
                    </div>
                  </div>
                  <t-table
                    row-key="assignment_key"
                    :data="classAnalyticsAssignments"
                    :columns="analyticsAssignmentColumns"
                    :pagination="{ pageSize: 8, total: classAnalyticsAssignments.length }"
                    size="small"
                  >
                    <template #assignment="{ row }">
                      <div class="resource-name-cell">
                        <strong>{{ analyticsAssignmentTitle(row) }}</strong>
                        <span>{{ row.assignment.group_id }}</span>
                      </div>
                    </template>
                    <template #started_count="{ row }">
                      {{ row.started_count }}/{{ classAnalytics.total_students }}
                    </template>
                    <template #completed_count="{ row }">
                      {{ row.completed_count }}/{{ classAnalytics.total_students }}
                    </template>
                    <template #completion_rate="{ row }">
                      {{ formatPercent(row.completion_rate) }}
                    </template>
                    <template #average_correct_rate="{ row }">
                      {{ formatPercent(row.average_correct_rate) }}
                    </template>
                    <template #due_at="{ row }">
                      {{ row.assignment.due_at ? formatDate(row.assignment.due_at) : '不限截止' }}
                    </template>
                  </t-table>
                  <t-empty v-if="!classAnalyticsAssignments.length && !analyticsLoading" size="small" description="暂无练习任务分析" />
                </div>
              </div>
              <t-empty v-else-if="!analyticsLoading" size="small" description="暂无班级分析数据" />
            </t-loading>
          </div>
        </t-tab-panel>
        <t-tab-panel value="settings" label="设置">
          <ClassSettingsPanel
            v-if="classInfo"
            :class-info="classInfo"
            :current-user-id="authStore.currentUserId"
            @updated="handleClassSettingsUpdated"
          />
        </t-tab-panel>
        <t-tab-panel
          v-for="item in futureTabs"
          :key="item.value"
          :value="item.value"
          :label="item.label"
          :disabled="isArchivedClass"
        >
          <div class="tab-panel">
            <h3>{{ item.label }}</h3>
            <p>{{ item.desc }}</p>
            <t-empty size="small" :description="item.empty" />
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-loading>

    <t-dialog
      v-model:visible="bindVisible"
      header="绑定知识库到班级"
      :confirm-btn="{ content: '绑定', loading: bindingResource }"
      @confirm="submitBindResource"
    >
      <t-form ref="resourceFormRef" :data="bindForm" :rules="bindRules" label-align="top">
        <t-form-item label="知识库" name="kb_id">
          <t-select
            v-model="bindForm.kb_id"
            :loading="knowledgeBasesLoading"
            placeholder="选择一个已创建的知识库"
            clearable
            filterable
          >
            <t-option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id" :label="kb.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="考试方向" name="domain_id">
          <t-select
            v-model="bindForm.domain_id"
            :loading="domainsLoading"
            placeholder="选择高考或雅思"
            clearable
            @change="handleBindDomainChange"
          >
            <t-option v-for="domain in domains" :key="domain.id" :value="domain.id" :label="domain.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="科目 / 模块" name="subject_id">
          <t-select
            v-model="bindForm.subject_id"
            :disabled="!bindForm.domain_id"
            :loading="subjectsLoading"
            placeholder="可选"
            clearable
          >
            <t-option v-for="subject in subjects" :key="subject.id" :value="subject.id" :label="subject.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="资料类型" name="material_type">
          <t-select v-model="bindForm.material_type">
            <t-option value="learning_material" label="学习资料" />
            <t-option value="exam_paper" label="试卷" />
            <t-option value="answer_key" label="答案" />
            <t-option value="explanation" label="解析" />
          </t-select>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="materialVisible"
      header="登记考试资料"
      width="720px"
      :confirm-btn="{ content: '登记', loading: registeringMaterial }"
      @confirm="submitRegisterMaterial"
    >
      <t-form ref="materialFormRef" :data="materialForm" :rules="materialRules" label-align="top">
        <div class="dialog-grid">
          <t-form-item label="知识库" name="knowledge_base_id">
            <t-select
              v-model="materialForm.knowledge_base_id"
              :loading="knowledgeBasesLoading"
              placeholder="选择资料所在知识库"
              clearable
              filterable
              @change="handleMaterialKBChange"
            >
              <t-option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id" :label="kb.name" />
            </t-select>
          </t-form-item>
          <t-form-item label="文档" name="knowledge_id">
            <t-select
              v-model="materialForm.knowledge_id"
              :disabled="!materialForm.knowledge_base_id"
              :loading="knowledgeFilesLoading"
              placeholder="选择具体试卷或学习资料"
              clearable
              filterable
            >
              <t-option
                v-for="file in knowledgeFiles"
                :key="file.id"
                :value="file.id"
                :label="knowledgeFileLabel(file)"
              />
            </t-select>
          </t-form-item>
        </div>
        <div class="dialog-grid">
          <t-form-item label="考试方向" name="domain_id">
            <t-select
              v-model="materialForm.domain_id"
              :loading="domainsLoading"
              placeholder="选择高考或雅思"
              clearable
              @change="handleMaterialDomainChange"
            >
              <t-option v-for="domain in domains" :key="domain.id" :value="domain.id" :label="domain.name" />
            </t-select>
          </t-form-item>
          <t-form-item label="科目 / 模块" name="subject_id">
            <t-select
              v-model="materialForm.subject_id"
              :disabled="!materialForm.domain_id"
              :loading="subjectsLoading"
              placeholder="可选"
              clearable
            >
              <t-option v-for="subject in subjects" :key="subject.id" :value="subject.id" :label="subject.name" />
            </t-select>
          </t-form-item>
        </div>
        <div class="dialog-grid">
          <t-form-item label="资料类型" name="material_type">
            <t-select v-model="materialForm.material_type">
              <t-option value="learning_material" label="学习资料" />
              <t-option value="exam_paper" label="试卷" />
              <t-option value="answer_key" label="答案" />
              <t-option value="explanation" label="解析" />
            </t-select>
          </t-form-item>
          <t-form-item label="关联题库" name="question_bank_id">
            <t-select
              v-model="materialForm.question_bank_id"
              :loading="questionBanksLoading"
              placeholder="留空则自动创建"
              clearable
              filterable
            >
              <t-option v-for="bank in questionBanks" :key="bank.id" :value="bank.id" :label="bank.name" />
            </t-select>
          </t-form-item>
        </div>
        <div class="dialog-grid">
          <t-form-item label="年份" name="source_year">
            <t-input-number v-model="materialForm.source_year" :min="1900" :max="2100" placeholder="可选" />
          </t-form-item>
          <t-form-item label="地区 / 套卷" name="source_region">
            <t-input v-model="materialForm.source_region" placeholder="如 全国甲卷 / Cambridge" />
          </t-form-item>
        </div>
        <t-form-item label="卷别 / 模块" name="paper_type">
          <t-input v-model="materialForm.paper_type" placeholder="如 英语阅读 / IELTS Reading" />
        </t-form-item>
        <t-form-item label="显示标题" name="title">
          <t-input v-model="materialForm.title" placeholder="留空则使用文档标题" />
        </t-form-item>
        <t-form-item label="说明" name="description">
          <t-textarea v-model="materialForm.description" placeholder="可选" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
        <t-form-item>
          <t-checkbox
            v-model="materialForm.create_task"
            :disabled="materialForm.material_type !== 'exam_paper'"
          >
            登记后创建结构化任务
          </t-checkbox>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="assignmentVisible"
      :header="assignmentDialogTitle"
      width="min(720px, calc(100vw - 16px))"
      :confirm-btn="{ content: assignmentConfirmLabel, loading: creatingAssignment }"
      @confirm="submitCreateAssignment"
    >
      <t-form ref="assignmentFormRef" class="assignment-form" :data="assignmentForm" :rules="assignmentRules" label-align="top">
        <t-form-item class="assignment-group-control" label="练习题组" name="group_id">
          <t-select
            v-if="assignmentMode === 'create'"
            v-model="assignmentForm.group_id"
            :loading="assignmentGroupsLoading"
            placeholder="选择一个已确认的正式题组"
            clearable
            filterable
          >
            <t-option
              v-for="item in assignmentGroups"
              :key="item.group.id"
              :value="item.group.id"
              :label="assignmentGroupOptionLabel(item)"
            />
          </t-select>
          <t-input
            v-else
            :value="editingAssignment ? assignmentGroupLabel(editingAssignment) : assignmentForm.group_id"
            readonly
          />
          <t-alert
            v-if="assignmentMode === 'create' && !assignmentGroupsLoading && !assignmentGroups.length"
            theme="info"
            message="暂无可发布题组，请先在题库中心完成试卷结构化并确认题组。"
          />
          <t-alert
            v-else-if="assignmentMode === 'create' && !assignmentGroupsLoading && hasImportableAssignmentGroups"
            theme="info"
            message="选择可导入题组发布时，系统会自动复制到当前班级空间后再生成练习任务。"
          />
        </t-form-item>
        <t-form-item label="任务标题" name="title">
          <t-input v-model="assignmentForm.title" placeholder="留空则使用题组标题" clearable />
        </t-form-item>
        <t-form-item label="截止时间" name="due_at">
          <t-input v-model="assignmentForm.due_at" placeholder="可选，例如 2026-07-30 23:59:00" clearable />
        </t-form-item>
        <t-form-item label="任务说明" name="instructions">
          <t-textarea
            v-model="assignmentForm.instructions"
            placeholder="可选：答题要求、复习范围或注意事项"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="assignmentProgressVisible"
      header="练习结果"
      width="920px"
      :footer="false"
    >
      <t-loading :loading="assignmentProgressLoading">
        <div v-if="assignmentProgressDetail" class="assignment-progress">
          <div class="assignment-progress__toolbar">
            <t-button
              size="small"
              variant="outline"
              :loading="remindingUserId === 'all'"
              :disabled="!!remindingUserId || !assignmentProgressRows.some(row => row.can_remind)"
              @click="sendProgressReminders()"
            >
              <template #icon><t-icon name="send" /></template>
              一键催交
            </t-button>
          </div>
          <div class="assignment-progress__summary">
            <div class="summary-item">
              <span>学生总数</span>
              <strong>{{ assignmentProgressDetail.total_students }}</strong>
            </div>
            <div class="summary-item">
              <span>已开始</span>
              <strong>{{ assignmentProgressDetail.started_count }}</strong>
            </div>
            <div class="summary-item">
              <span>已完成</span>
              <strong>{{ assignmentProgressDetail.completed_count }}</strong>
            </div>
            <div class="summary-item">
              <span>平均正确率</span>
              <strong>{{ formatPercent(assignmentProgressDetail.average_correct_rate) }}</strong>
            </div>
          </div>
          <t-table
            row-key="user_id"
            :data="assignmentProgressRows"
            :columns="assignmentProgressColumns"
            :pagination="{ pageSize: 8, total: assignmentProgressRows.length }"
            size="small"
          >
            <template #student="{ row }">
              <div class="resource-name-cell">
                <strong>{{ row.display_name }}</strong>
                <span>{{ row.display_id }}</span>
              </div>
            </template>
            <template #status="{ row }">
              <t-tag variant="light" :theme="assignmentProgressStatusTheme(row.status)">
                {{ assignmentProgressStatusLabel(row.status) }}
              </t-tag>
            </template>
            <template #correct_rate="{ row }">
              {{ formatPercent(row.correct_rate) }}
            </template>
            <template #completed_at="{ row }">
              {{ row.completed_at ? formatDate(row.completed_at) : '-' }}
            </template>
            <template #reminder="{ row }">
              <t-tooltip v-if="row.status !== 'completed'" :content="assignmentReminderTooltip(row)">
                <t-button
                  shape="square"
                  variant="text"
                  size="small"
                  :loading="remindingUserId === row.user_id"
                  :disabled="!!remindingUserId || !row.can_remind"
                  aria-label="催交该学生"
                  @click="sendProgressReminders(row.user_id)"
                >
                  <t-icon name="send" />
                </t-button>
              </t-tooltip>
              <span v-else class="muted-text">-</span>
            </template>
          </t-table>
        </div>
      </t-loading>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { approveExamClassMember, getExamClass, listExamClassMembers, rejectExamClassMember } from '@/api/exam/class'
import { listExamDomains, listExamSubjects } from '@/api/exam/domain'
import { listExamMaterials, listExamStructuringTasks, registerExamMaterial } from '@/api/exam/material'
import { extractQuestionGroupDrafts } from '@/api/exam/question-group-draft'
import { listQuestionBanks } from '@/api/exam/question-bank'
import { bindKnowledgeBaseResource, listExamResources } from '@/api/exam/resource'
import { getClassAnalytics } from '@/api/exam/analytics'
import {
  createAssignmentAttempt,
  createClassAssignment,
  getClassAssignmentProgress,
  listClassAssignments,
  republishClassAssignment,
  updateClassAssignment,
  withdrawClassAssignment,
} from '@/api/exam/assignment'
import { sendAssignmentReminders } from '@/api/exam/assignmentNotification'
import { listPracticeQuestionGroups } from '@/api/exam/practice'
import { listKnowledgeBases, listKnowledgeFiles } from '@/api/knowledge-base'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import ClassPracticeRecommendations from './ClassPracticeRecommendations.vue'
import ClassSettingsPanel from './ClassSettingsPanel.vue'
import {
  assignmentLifecycleState,
  assignmentStatusLabel,
  canCreateAssignmentAttempt,
  canEditAssignment,
  canRepublishAssignment,
  canWithdrawAssignment,
  existingAssignmentAttemptID,
} from './assignmentLifecycle'
import { memberDisplayId, memberDisplayName } from './memberDisplay'
import { isArchivedExamClass } from './classSettings'
import type {
  ExamClassAssignment,
  ExamClass,
  ExamClassAnalyticsAssignment,
  ExamClassAnalyticsSummary,
  ExamPracticeRecommendation,
  ExamAssignmentProgressStatus,
  ExamAssignmentProgressSummary,
  ExamAssignmentSummary,
  ExamClassMember,
  ExamClassRole,
  ExamDomain,
  ExamMaterial,
  ExamMaterialIngestStatus,
  ExamMaterialType,
  ExamStructuringTask,
  ExamStructuringTaskStatus,
  ExamSpaceResource,
  ExamSubject,
  QuestionBank,
  QuestionGroupPracticeSummary,
  ReviewStatus,
} from '@/types/exam'
import type { KnowledgeBaseInfo } from '@/api/auth'

interface KnowledgeFileItem {
  id: string
  title?: string
  file_name?: string
  source?: string
  parse_status?: string
}

interface AssignmentProgressRow {
  user_id: string
  display_id: string
  display_name: string
  status: ExamAssignmentProgressStatus
  answered_text: string
  correct_text: string
  correct_rate: number
  completed_at?: string
  last_reminded_at?: string
  can_remind: boolean
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const loading = ref(false)
const membersLoading = ref(false)
const resourcesLoading = ref(false)
const materialsLoading = ref(false)
const assignmentsLoading = ref(false)
const analyticsLoading = ref(false)
const assignmentGroupsLoading = ref(false)
const assignmentProgressLoading = ref(false)
const domainsLoading = ref(false)
const subjectsLoading = ref(false)
const knowledgeBasesLoading = ref(false)
const knowledgeFilesLoading = ref(false)
const questionBanksLoading = ref(false)
const bindingResource = ref(false)
const registeringMaterial = ref(false)
const creatingAssignment = ref(false)
const reviewingUserId = ref('')
const extractingTaskId = ref('')
const startingAssignmentId = ref('')
const loadingAssignmentProgressId = ref('')
const assignmentProgressAssignmentId = ref('')
const remindingUserId = ref('')
const assignmentActionId = ref('')
const activeTab = ref('overview')
const classInfo = ref<ExamClass | null>(null)
const members = ref<ExamClassMember[]>([])
const resources = ref<ExamSpaceResource[]>([])
const materials = ref<ExamMaterial[]>([])
const structuringTasks = ref<ExamStructuringTask[]>([])
const domains = ref<ExamDomain[]>([])
const subjects = ref<ExamSubject[]>([])
const knowledgeBases = ref<KnowledgeBaseInfo[]>([])
const knowledgeFiles = ref<KnowledgeFileItem[]>([])
const questionBanks = ref<QuestionBank[]>([])
const assignments = ref<ExamAssignmentSummary[]>([])
const assignmentGroups = ref<QuestionGroupPracticeSummary[]>([])
const classAnalytics = ref<ExamClassAnalyticsSummary | null>(null)
const assignmentProgressDetail = ref<ExamAssignmentProgressSummary | null>(null)
const resourceFormRef = ref<FormInstanceFunctions>()
const materialFormRef = ref<FormInstanceFunctions>()
const assignmentFormRef = ref<FormInstanceFunctions>()
const bindVisible = ref(false)
const materialVisible = ref(false)
const assignmentVisible = ref(false)
const assignmentProgressVisible = ref(false)
const assignmentMode = ref<'create' | 'edit'>('create')
const editingAssignment = ref<ExamAssignmentSummary | null>(null)
const canReviewMembers = computed(() => authStore.hasRole('contributor'))
const canManageResources = computed(() => authStore.hasRole('contributor'))
const canManageAssignments = computed(() => authStore.hasRole('contributor'))
const canViewAnalytics = computed(() => authStore.hasRole('contributor'))
const assignmentDialogTitle = computed(() => assignmentMode.value === 'edit' ? '编辑练习任务' : '发布练习任务')
const assignmentConfirmLabel = computed(() => assignmentMode.value === 'edit' ? '保存' : '发布')
const currentClassId = computed(() => String(route.params.classId || ''))
const isArchivedClass = computed(() => isArchivedExamClass(classInfo.value))
const hasImportableAssignmentGroups = computed(() => {
  const classSpaceId = classInfo.value?.space_id
  return assignmentGroups.value.some((item) => item.group?.space_id && item.group.space_id !== classSpaceId)
})
const classAnalyticsMembers = computed(() => {
  return (classAnalytics.value?.members || []).map(item => ({
    ...item,
    student_key: item.member.user_id,
  }))
})
const classAnalyticsAssignments = computed(() => {
  return (classAnalytics.value?.assignments || []).map(item => ({
    ...item,
    assignment_key: item.assignment.id,
  }))
})
const assignmentProgressRows = computed<AssignmentProgressRow[]>(() => {
  return (assignmentProgressDetail.value?.members || []).map((item) => {
    const attempt = item.attempt
    return {
      user_id: item.member.user_id,
      display_id: memberDisplayId(item.member),
      display_name: memberDisplayName(item.member),
      status: item.status,
      answered_text: attempt ? `${attempt.answered_count}/${attempt.question_count}` : '-',
      correct_text: attempt ? `${attempt.correct_count}/${attempt.question_count}` : '-',
      correct_rate: item.correct_rate || 0,
      completed_at: attempt?.completed_at,
      last_reminded_at: item.last_reminded_at,
      can_remind: item.can_remind,
    }
  })
})

const bindForm = ref({
  kb_id: '',
  domain_id: '',
  subject_id: '',
  material_type: 'learning_material' as ExamMaterialType,
})

const materialForm = ref<{
  knowledge_base_id: string
  knowledge_id: string
  domain_id: string
  subject_id: string
  material_type: ExamMaterialType
  title: string
  description: string
  source_year?: number
  source_region: string
  paper_type: string
  question_bank_id: string
  create_task: boolean
}>({
  knowledge_base_id: '',
  knowledge_id: '',
  domain_id: '',
  subject_id: '',
  material_type: 'exam_paper',
  title: '',
  description: '',
  source_region: '',
  paper_type: '',
  question_bank_id: '',
  create_task: true,
})

const assignmentForm = ref({
  group_id: '',
  title: '',
  instructions: '',
  due_at: '',
})

const futureTabs = [
  { value: 'questionSets', label: '题集', desc: '班级题集来自题库筛选、试卷结构化和老师手动组题。', empty: '题集能力将在结构化题库后启用' },
  { value: 'entitlements', label: '权益', desc: '高成本解析、Agent 工具调用和班级人数会进入权益校验。', empty: '权益明细将在支付模块接入后显示' },
]

const memberColumns = [
  { colKey: 'member', title: '姓名', cell: 'member', ellipsis: true },
  { colKey: 'role', title: '班级角色', cell: 'role', width: 120 },
  { colKey: 'status', title: '状态', cell: 'status', width: 120 },
  { colKey: 'created_at', title: '申请时间', cell: 'created_at', width: 160 },
  { colKey: 'actions', title: '操作', cell: 'actions', width: 160 },
]

const resourceColumns = [
  { colKey: 'resource_id', title: '知识库', cell: 'resource_id', ellipsis: true },
  { colKey: 'material_type', title: '资料类型', cell: 'material_type', width: 120 },
  { colKey: 'domain_id', title: '考试方向', cell: 'domain_id', width: 120 },
  { colKey: 'review_status', title: '状态', cell: 'review_status', width: 120 },
  { colKey: 'updated_at', title: '更新时间', cell: 'updated_at', width: 160 },
  { colKey: 'actions', title: '操作', cell: 'actions', width: 120 },
]

const materialColumns = [
  { colKey: 'title', title: '资料', cell: 'title', ellipsis: true },
  { colKey: 'material_type', title: '类型', cell: 'material_type', width: 100 },
  { colKey: 'ingest_status', title: '入库状态', cell: 'ingest_status', width: 120 },
  { colKey: 'domain_id', title: '考试方向', cell: 'domain_id', width: 120 },
  { colKey: 'source_year', title: '年份', cell: 'source_year', width: 90 },
  { colKey: 'updated_at', title: '更新时间', cell: 'updated_at', width: 160 },
]

const taskColumns = [
  { colKey: 'material_id', title: '来源资料', cell: 'material_id', ellipsis: true },
  { colKey: 'status', title: '任务状态', cell: 'status', width: 130 },
  { colKey: 'source_chunk_count', title: 'chunk', width: 90 },
  { colKey: 'structured_question_count', title: '已入题', width: 90 },
  { colKey: 'created_at', title: '创建时间', cell: 'created_at', width: 160 },
  { colKey: 'actions', title: '操作', cell: 'actions', width: 180 },
]

const assignmentProgressColumns = [
  { colKey: 'student', title: '学生', cell: 'student', ellipsis: true },
  { colKey: 'status', title: '状态', cell: 'status', width: 110 },
  { colKey: 'answered_text', title: '答题数', width: 100 },
  { colKey: 'correct_text', title: '正确数', width: 100 },
  { colKey: 'correct_rate', title: '正确率', cell: 'correct_rate', width: 100 },
  { colKey: 'completed_at', title: '完成时间', cell: 'completed_at', width: 180 },
  { colKey: 'reminder', title: '催交', cell: 'reminder', width: 72 },
]

const analyticsMemberColumns = [
  { colKey: 'student', title: '学生', cell: 'student', ellipsis: true },
  { colKey: 'started_count', title: '已开始', width: 100 },
  { colKey: 'progress', title: '完成进度', cell: 'progress', width: 110 },
  { colKey: 'completion_rate', title: '完成率', cell: 'completion_rate', width: 100 },
  { colKey: 'average_correct_rate', title: '平均正确率', cell: 'average_correct_rate', width: 120 },
  { colKey: 'last_activity_at', title: '最近练习', cell: 'last_activity_at', width: 180 },
]

const analyticsAssignmentColumns = [
  { colKey: 'assignment', title: '任务', cell: 'assignment', ellipsis: true },
  { colKey: 'started_count', title: '已开始', cell: 'started_count', width: 100 },
  { colKey: 'completed_count', title: '已完成', cell: 'completed_count', width: 100 },
  { colKey: 'completion_rate', title: '完成率', cell: 'completion_rate', width: 100 },
  { colKey: 'average_correct_rate', title: '平均正确率', cell: 'average_correct_rate', width: 120 },
  { colKey: 'due_at', title: '截止时间', cell: 'due_at', width: 180 },
]

const bindRules: Record<string, FormRule[]> = {
  kb_id: [{ required: true, message: '请选择知识库', type: 'error' }],
  domain_id: [{ required: true, message: '请选择考试方向', type: 'error' }],
}

const materialRules: Record<string, FormRule[]> = {
  knowledge_base_id: [{ required: true, message: '请选择知识库', type: 'error' }],
  knowledge_id: [{ required: true, message: '请选择文档', type: 'error' }],
  domain_id: [{ required: true, message: '请选择考试方向', type: 'error' }],
}

const assignmentRules = computed<Record<string, FormRule[]>>(() => ({
  group_id: [{ required: true, message: '请选择题组', type: 'error' }],
  ...(assignmentMode.value === 'edit'
    ? { title: [{ required: true, message: '请输入任务标题', type: 'error' }] }
    : {}),
}))

const formatDate = (value?: string) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

const formatPercent = (value?: number) => {
  if (!value) return '0%'
  return `${Math.round(value * 100)}%`
}

const assignmentProgressStatusLabel = (status: ExamAssignmentProgressStatus) => {
  const map: Record<ExamAssignmentProgressStatus, string> = {
    not_started: '未开始',
    in_progress: '进行中',
    completed: '已完成',
  }
  return map[status] || status
}

const assignmentProgressStatusTheme = (status: ExamAssignmentProgressStatus) => {
  if (status === 'completed') return 'success'
  if (status === 'in_progress') return 'warning'
  return 'default'
}

const domainName = (domainId?: string) => {
  if (!domainId) return '未绑定'
  return domains.value.find(item => item.id === domainId)?.name || '未知考试域'
}

const materialTypeLabel = (type: ExamMaterialType) => {
  const map: Record<ExamMaterialType, string> = {
    learning_material: '学习资料',
    exam_paper: '试卷',
    answer_key: '答案',
    explanation: '解析',
  }
  return map[type] || type
}

const reviewLabel = (status: ReviewStatus) => {
  const map: Record<ReviewStatus, string> = {
    private: '班级可见',
    pending: '待审核',
    approved: '已公开',
    rejected: '已驳回',
  }
  return map[status] || status
}

const reviewTag = (status: ReviewStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

const ingestStatusLabel = (status: ExamMaterialIngestStatus) => {
  const map: Record<ExamMaterialIngestStatus, string> = {
    pending: '待解析',
    processing: '解析中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
    unknown: '未知',
  }
  return map[status] || status
}

const ingestStatusTheme = (status: ExamMaterialIngestStatus) => {
  if (status === 'completed') return 'success'
  if (status === 'processing' || status === 'pending') return 'warning'
  if (status === 'failed' || status === 'cancelled') return 'danger'
  return 'default'
}

const taskStatusLabel = (status: ExamStructuringTaskStatus) => {
  const map: Record<ExamStructuringTaskStatus, string> = {
    pending: '待处理',
    ready_for_review: '待校对',
    blocked: '已阻塞',
    extracting: '抽题中',
    reviewing: '校对中',
    completed: '已完成',
    failed: '失败',
  }
  return map[status] || status
}

const taskStatusTheme = (status: ExamStructuringTaskStatus) => {
  if (status === 'ready_for_review' || status === 'completed') return 'success'
  if (status === 'pending' || status === 'extracting' || status === 'reviewing') return 'warning'
  if (status === 'blocked' || status === 'failed') return 'danger'
  return 'default'
}

const knowledgeBaseName = (kbId: string) => {
  return knowledgeBases.value.find(item => item.id === kbId)?.name || '未知知识库'
}

const knowledgeFileLabel = (file: KnowledgeFileItem) => {
  const title = file.file_name || file.title || file.source || file.id
  return file.parse_status ? `${title}（${file.parse_status}）` : title
}

const taskMaterialTitle = (materialId: string) => {
  return materials.value.find(item => item.id === materialId)?.title || materialId
}

const roleLabel = (role: ExamClassRole) => {
  if (role === 'teacher') return '老师'
  if (role === 'assistant') return '助教'
  return '学生'
}

const roleTheme = (role: ExamClassRole) => {
  if (role === 'teacher') return 'primary'
  if (role === 'assistant') return 'warning'
  return 'default'
}

const loadData = async () => {
  const classId = String(route.params.classId || '')
  if (!classId) return
  loading.value = true
  try {
    const res = await getExamClass(classId)
    classInfo.value = res.data
    if (isArchivedClass.value && archivedBusinessTabs.has(activeTab.value)) {
      activeTab.value = 'settings'
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级详情加载失败')
  } finally {
    loading.value = false
  }
}

const archivedBusinessTabs = new Set(['members', 'resources', 'assignments', 'analytics', 'questionSets', 'entitlements'])

const handleClassSettingsUpdated = (updated: ExamClass) => {
  classInfo.value = updated
  if (updated.status === 'archived') {
    activeTab.value = 'settings'
  }
}

const loadDomains = async () => {
  domainsLoading.value = true
  try {
    const res = await listExamDomains()
    domains.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '考试方向加载失败')
  } finally {
    domainsLoading.value = false
  }
}

const loadSubjects = async (domainId: string) => {
  subjects.value = []
  if (!domainId) return
  subjectsLoading.value = true
  try {
    const res = await listExamSubjects(domainId)
    subjects.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '科目列表加载失败')
  } finally {
    subjectsLoading.value = false
  }
}

const loadKnowledgeBases = async () => {
  knowledgeBasesLoading.value = true
  try {
    const res: any = await listKnowledgeBases()
    knowledgeBases.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库列表加载失败')
  } finally {
    knowledgeBasesLoading.value = false
  }
}

const loadKnowledgeFiles = async (kbId: string) => {
  knowledgeFiles.value = []
  if (!kbId) return
  knowledgeFilesLoading.value = true
  try {
    const res: any = await listKnowledgeFiles(kbId, { page: 1, page_size: 100 })
    knowledgeFiles.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '文档列表加载失败')
  } finally {
    knowledgeFilesLoading.value = false
  }
}

const loadQuestionBanks = async () => {
  if (!classInfo.value?.space_id) return
  questionBanksLoading.value = true
  try {
    const res = await listQuestionBanks({ space_id: classInfo.value.space_id })
    questionBanks.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库列表加载失败')
  } finally {
    questionBanksLoading.value = false
  }
}

const loadMembers = async () => {
  if (!canReviewMembers.value) return
  const classId = String(route.params.classId || '')
  if (!classId) return
  membersLoading.value = true
  try {
    const res = await listExamClassMembers(classId)
    members.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '成员列表加载失败')
  } finally {
    membersLoading.value = false
  }
}

const loadResources = async () => {
  if (!classInfo.value?.space_id) return
  resourcesLoading.value = true
  try {
    const [resourceRes] = await Promise.all([
      listExamResources({ space_id: classInfo.value.space_id, resource_type: 'knowledge_base' }),
      knowledgeBases.value.length ? Promise.resolve(null) : loadKnowledgeBases(),
      domains.value.length ? Promise.resolve(null) : loadDomains(),
    ])
    resources.value = resourceRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级资料加载失败')
  } finally {
    resourcesLoading.value = false
  }
}

const loadMaterials = async () => {
  if (!classInfo.value?.space_id) return
  materialsLoading.value = true
  try {
    const [materialsRes, tasksRes] = await Promise.all([
      listExamMaterials({ space_id: classInfo.value.space_id }),
      listExamStructuringTasks({ space_id: classInfo.value.space_id }),
    ])
    materials.value = materialsRes.data || []
    structuringTasks.value = tasksRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '考试资料加载失败')
  } finally {
    materialsLoading.value = false
  }
}

const loadResourceTab = async () => {
  await Promise.all([
    loadResources(),
    loadMaterials(),
  ])
}

const loadAssignments = async () => {
  const classId = String(route.params.classId || '')
  if (!classId) return
  assignmentsLoading.value = true
  try {
    const res = await listClassAssignments(classId, { limit: 100 })
    assignments.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级练习任务加载失败')
  } finally {
    assignmentsLoading.value = false
  }
}

const loadClassAnalytics = async () => {
  if (!canViewAnalytics.value) return
  const classId = String(route.params.classId || '')
  if (!classId) return
  analyticsLoading.value = true
  try {
    const res = await getClassAnalytics(classId)
    classAnalytics.value = res.data
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级分析加载失败')
  } finally {
    analyticsLoading.value = false
  }
}

const loadAssignmentGroups = async () => {
  if (!classInfo.value?.space_id) return
  assignmentGroupsLoading.value = true
  try {
    const res = await listPracticeQuestionGroups({ limit: 100 })
    assignmentGroups.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '可发布题组加载失败')
  } finally {
    assignmentGroupsLoading.value = false
  }
}

const assignmentGroupTypeLabel = (type?: string) => {
  const map: Record<string, string> = {
    reading_passage: '阅读',
    math_problem: '数学题',
    single_question: '单题',
    cloze: '完型',
    essay: '作文',
  }
  return type ? (map[type] || type) : '题组'
}

const assignmentGroupOptionLabel = (item: QuestionGroupPracticeSummary) => {
  const title = item.group.title || item.group.material_text || assignmentGroupTypeLabel(item.group.group_type)
  const scope = item.group.space_id === classInfo.value?.space_id ? '班级题组' : '可导入题组'
  return `${title} · ${scope} · ${item.bank_name || '题库'} · ${item.question_count} 题`
}

const assignmentGroupLabel = (item: ExamAssignmentSummary) => {
  return item.group?.title || item.group?.material_text || assignmentGroupTypeLabel(item.group?.group_type)
}

const analyticsAssignmentTitle = (item: ExamClassAnalyticsAssignment) => {
  return item.assignment.title || item.assignment.group_id || '未命名任务'
}

const assignmentProgress = (item: ExamAssignmentSummary) => {
  const attempt = item.last_attempt
  if (!attempt) return '未开始'
  if (attempt.status === 'completed') return `已完成 ${attempt.correct_count}/${attempt.question_count}`
  return `进行中 ${attempt.answered_count}/${attempt.question_count}`
}

const assignmentStatusTheme = (assignment: ExamClassAssignment) => {
  const state = assignmentLifecycleState(assignment)
  if (state === 'active') return 'success'
  if (state === 'expired') return 'warning'
  return 'default'
}

const assignmentStartLabel = (item: ExamAssignmentSummary) => {
  if (existingAssignmentAttemptID(item)) return '继续练习'
  return assignmentLifecycleState(item.assignment) === 'expired' ? '已截止' : '开始练习'
}

const assignmentDueText = (value?: string) => {
  if (!value) return '不限截止'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return `截止 ${value}`
  return `截止 ${date.toLocaleString()}`
}

const normalizeAssignmentDueAt = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  let normalized = trimmed.includes('T') ? trimmed : trimmed.replace(' ', 'T')
  if (normalized.length === 10) {
    normalized = `${normalized}T23:59:59`
  }
  if (!/[zZ]|[+-]\d{2}:\d{2}$/.test(normalized)) {
    normalized = `${normalized}+08:00`
  }
  return normalized
}

const assignmentDueInput = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const openAssignmentDialog = async () => {
  if (!canManageAssignments.value) return
  assignmentMode.value = 'create'
  editingAssignment.value = null
  assignmentForm.value = {
    group_id: '',
    title: '',
    instructions: '',
    due_at: '',
  }
  assignmentVisible.value = true
  await loadAssignmentGroups()
}

const openEditAssignmentDialog = (item: ExamAssignmentSummary) => {
  if (!canManageAssignments.value || !canEditAssignment(item.assignment)) return
  assignmentMode.value = 'edit'
  editingAssignment.value = item
  assignmentForm.value = {
    group_id: item.assignment.group_id,
    title: item.assignment.title,
    instructions: item.assignment.instructions || '',
    due_at: assignmentDueInput(item.assignment.due_at),
  }
  assignmentVisible.value = true
}

const prefillRecommendedAssignment = async (recommendation: ExamPracticeRecommendation) => {
  if (!canManageAssignments.value || !recommendation.group?.group?.id) return
  assignmentMode.value = 'create'
  editingAssignment.value = null
  await loadAssignmentGroups()
  const groupID = recommendation.group.group.id
  if (!assignmentGroups.value.some(item => item.group.id === groupID)) {
    assignmentGroups.value.unshift(recommendation.group)
  }
  assignmentForm.value.group_id = groupID
  assignmentForm.value.title = recommendation.group.group.title || '推荐练习'
  assignmentForm.value.instructions = '根据班级学习诊断推荐，请确认题组内容和截止时间后发布。'
  assignmentForm.value.due_at = ''
  assignmentVisible.value = true
}

const submitCreateAssignment = async () => {
  const classId = String(route.params.classId || '')
  if (!classId) return
  const result = await assignmentFormRef.value?.validate()
  if (result !== true) return

  creatingAssignment.value = true
  try {
    if (assignmentMode.value === 'edit' && editingAssignment.value) {
      await updateClassAssignment(classId, editingAssignment.value.assignment.id, {
        title: assignmentForm.value.title.trim(),
        instructions: assignmentForm.value.instructions.trim(),
        due_at: normalizeAssignmentDueAt(assignmentForm.value.due_at) || null,
      })
      MessagePlugin.success('练习任务已更新')
    } else {
      await createClassAssignment(classId, {
        group_id: assignmentForm.value.group_id,
        title: assignmentForm.value.title.trim() || undefined,
        instructions: assignmentForm.value.instructions.trim() || undefined,
        due_at: normalizeAssignmentDueAt(assignmentForm.value.due_at),
      })
      MessagePlugin.success('练习任务已发布')
    }
    assignmentVisible.value = false
    await loadAssignments()
  } catch (error: any) {
    MessagePlugin.error(error?.message || (assignmentMode.value === 'edit' ? '更新练习任务失败' : '发布练习任务失败'))
    if (error?.status === 409) await loadAssignments()
  } finally {
    creatingAssignment.value = false
  }
}

const confirmWithdrawAssignment = (item: ExamAssignmentSummary) => {
  const classId = String(route.params.classId || '')
  if (!classId || !canWithdrawAssignment(item.assignment)) return
  const confirmDialog = DialogPlugin.confirm({
    header: '撤回练习任务',
    body: '学生入口将隐藏，历史答题不会删除。',
    confirmBtn: { content: '确认撤回', theme: 'danger' },
    theme: 'warning',
    onConfirm: async () => {
      assignmentActionId.value = item.assignment.id
      try {
        await withdrawClassAssignment(classId, item.assignment.id)
        MessagePlugin.success('练习任务已撤回')
        await loadAssignments()
        confirmDialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '撤回练习任务失败')
        if (error?.status === 409) await loadAssignments()
      } finally {
        assignmentActionId.value = ''
      }
    },
  })
}

const submitRepublishAssignment = async (item: ExamAssignmentSummary) => {
  const classId = String(route.params.classId || '')
  if (!classId || !canRepublishAssignment(item.assignment)) return
  assignmentActionId.value = item.assignment.id
  try {
    await republishClassAssignment(classId, item.assignment.id)
    MessagePlugin.success('练习任务已重新发布')
    await loadAssignments()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '重新发布练习任务失败')
    if (error?.status === 409) await loadAssignments()
  } finally {
    assignmentActionId.value = ''
  }
}

const loadAssignmentProgress = async () => {
  const classId = currentClassId.value
  const assignmentId = assignmentProgressAssignmentId.value
  if (!classId || !assignmentId) return
  assignmentProgressLoading.value = true
  loadingAssignmentProgressId.value = assignmentId
  try {
    const res = await getClassAssignmentProgress(classId, assignmentId)
    assignmentProgressDetail.value = res.data
  } catch (error: any) {
    MessagePlugin.error(error?.message || '练习结果加载失败')
  } finally {
    assignmentProgressLoading.value = false
    loadingAssignmentProgressId.value = ''
  }
}

const openAssignmentProgress = async (item: ExamAssignmentSummary) => {
  if (!currentClassId.value || !item.assignment?.id || !canManageAssignments.value) return
  assignmentProgressVisible.value = true
  assignmentProgressDetail.value = null
  assignmentProgressAssignmentId.value = item.assignment.id
  await loadAssignmentProgress()
}

const sendProgressReminders = async (recipientUserId?: string) => {
  const classId = currentClassId.value
  const assignmentId = assignmentProgressAssignmentId.value
  if (!classId || !assignmentId || remindingUserId.value) return
  remindingUserId.value = recipientUserId || 'all'
  try {
    const res = await sendAssignmentReminders(classId, assignmentId, {
      recipient_user_ids: recipientUserId ? [recipientUserId] : [],
    })
    const result = res.data
    MessagePlugin.success(
      `已发送 ${result.sent_count} 条提醒，完成跳过 ${result.completed_skipped_count}，限频跳过 ${result.cooldown_skipped_count}`,
    )
    await loadAssignmentProgress()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '催交失败')
  } finally {
    remindingUserId.value = ''
  }
}

const assignmentReminderTooltip = (row: AssignmentProgressRow) => {
  if (row.can_remind) return '催交该学生'
  return row.last_reminded_at ? '24 小时内已催' : '当前不可催交'
}

const startAssignmentPractice = async (item: ExamAssignmentSummary) => {
  if (!item.assignment?.id) return
  const existingAttemptId = existingAssignmentAttemptID(item)
  if (existingAttemptId) {
    router.push(`/platform/practice/question-groups/${item.assignment.group_id}?attempt_id=${existingAttemptId}`)
    return
  }
  if (!canCreateAssignmentAttempt(item)) return
  startingAssignmentId.value = item.assignment.id
  try {
    const res = await createAssignmentAttempt(item.assignment.id)
    const attemptId = res.data?.attempt?.id
    const query = attemptId ? `?attempt_id=${attemptId}` : ''
    router.push(`/platform/practice/question-groups/${item.assignment.group_id}${query}`)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '进入练习任务失败')
  } finally {
    startingAssignmentId.value = ''
  }
}

const extractTask = async (task: ExamStructuringTask) => {
  if (!task?.id) return
  extractingTaskId.value = task.id
  try {
    await extractQuestionGroupDrafts(task.id, task.status === 'failed')
    MessagePlugin.success('题组抽取任务已启动')
    router.push(`/platform/question-group-drafts/${task.id}`)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题组抽取失败')
  } finally {
    extractingTaskId.value = ''
  }
}

const approveMember = async (userId: string) => {
  const classId = String(route.params.classId || '')
  if (!classId || !userId) return
  reviewingUserId.value = userId
  try {
    await approveExamClassMember(classId, userId)
    MessagePlugin.success('已通过加入申请')
    await loadMembers()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '审核失败')
  } finally {
    reviewingUserId.value = ''
  }
}

const rejectMember = async (userId: string) => {
  const classId = String(route.params.classId || '')
  if (!classId || !userId) return
  reviewingUserId.value = userId
  try {
    await rejectExamClassMember(classId, userId)
    MessagePlugin.success('已拒绝加入申请')
    await loadMembers()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '审核失败')
  } finally {
    reviewingUserId.value = ''
  }
}

const openBindDialog = async () => {
  if (!classInfo.value?.space_id) return
  bindForm.value = {
    kb_id: '',
    domain_id: classInfo.value.domain_id || domains.value[0]?.id || '',
    subject_id: '',
    material_type: 'learning_material',
  }
  bindVisible.value = true
  await Promise.all([
    knowledgeBases.value.length ? Promise.resolve(null) : loadKnowledgeBases(),
    domains.value.length ? Promise.resolve(null) : loadDomains(),
  ])
  if (!bindForm.value.domain_id) {
    bindForm.value.domain_id = classInfo.value?.domain_id || domains.value[0]?.id || ''
  }
  if (bindForm.value.domain_id) {
    await loadSubjects(bindForm.value.domain_id)
  }
}

const handleBindDomainChange = async (value: string | number | boolean) => {
  const domainId = typeof value === 'string' ? value : ''
  bindForm.value.subject_id = ''
  await loadSubjects(domainId)
}

const openMaterialDialog = async () => {
  if (!classInfo.value?.space_id) return
  materialForm.value = {
    knowledge_base_id: resources.value[0]?.resource_id || '',
    knowledge_id: '',
    domain_id: classInfo.value.domain_id || domains.value[0]?.id || '',
    subject_id: '',
    material_type: 'exam_paper',
    title: '',
    description: '',
    source_region: '',
    paper_type: '',
    question_bank_id: '',
    create_task: true,
  }
  materialVisible.value = true
  await Promise.all([
    knowledgeBases.value.length ? Promise.resolve(null) : loadKnowledgeBases(),
    domains.value.length ? Promise.resolve(null) : loadDomains(),
    resources.value.length ? Promise.resolve(null) : loadResources(),
    loadQuestionBanks(),
  ])
  if (!materialForm.value.knowledge_base_id) {
    materialForm.value.knowledge_base_id = resources.value[0]?.resource_id || knowledgeBases.value[0]?.id || ''
  }
  if (!materialForm.value.domain_id) {
    materialForm.value.domain_id = classInfo.value?.domain_id || domains.value[0]?.id || ''
  }
  await Promise.all([
    materialForm.value.knowledge_base_id ? loadKnowledgeFiles(materialForm.value.knowledge_base_id) : Promise.resolve(null),
    materialForm.value.domain_id ? loadSubjects(materialForm.value.domain_id) : Promise.resolve(null),
  ])
}

const handleMaterialKBChange = async (value: string | number | boolean) => {
  const kbId = typeof value === 'string' ? value : ''
  materialForm.value.knowledge_id = ''
  await loadKnowledgeFiles(kbId)
}

const handleMaterialDomainChange = async (value: string | number | boolean) => {
  const domainId = typeof value === 'string' ? value : ''
  materialForm.value.subject_id = ''
  await loadSubjects(domainId)
}

const submitBindResource = async () => {
  if (!classInfo.value?.space_id) return
  const result = await resourceFormRef.value?.validate()
  if (result !== true) return

  bindingResource.value = true
  try {
    await bindKnowledgeBaseResource(bindForm.value.kb_id, {
      space_id: classInfo.value.space_id,
      domain_id: bindForm.value.domain_id,
      subject_id: bindForm.value.subject_id || undefined,
      material_type: bindForm.value.material_type,
    })
    MessagePlugin.success('知识库已绑定到班级')
    bindVisible.value = false
    await loadResourceTab()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '绑定知识库失败')
  } finally {
    bindingResource.value = false
  }
}

const submitRegisterMaterial = async () => {
  if (!classInfo.value?.space_id) return
  const result = await materialFormRef.value?.validate()
  if (result !== true) return

  registeringMaterial.value = true
  try {
    await registerExamMaterial({
      space_id: classInfo.value.space_id,
      knowledge_base_id: materialForm.value.knowledge_base_id,
      knowledge_id: materialForm.value.knowledge_id,
      domain_id: materialForm.value.domain_id,
      subject_id: materialForm.value.subject_id || undefined,
      material_type: materialForm.value.material_type,
      title: materialForm.value.title || undefined,
      description: materialForm.value.description || undefined,
      source_year: materialForm.value.source_year || undefined,
      source_region: materialForm.value.source_region || undefined,
      paper_type: materialForm.value.paper_type || undefined,
      question_bank_id: materialForm.value.question_bank_id || undefined,
      create_task: materialForm.value.material_type === 'exam_paper' && materialForm.value.create_task,
    })
    MessagePlugin.success('考试资料已登记')
    materialVisible.value = false
    await Promise.all([
      loadResourceTab(),
      loadQuestionBanks(),
    ])
  } catch (error: any) {
    MessagePlugin.error(error?.message || '登记考试资料失败')
  } finally {
    registeringMaterial.value = false
  }
}

const startChat = (kbId: string) => {
  if (!kbId) return
  settingsStore.selectKnowledgeBases([kbId])
  settingsStore.clearFiles()
  settingsStore.clearTags()
  router.push('/platform/creatChat')
}

watch(activeTab, (tab) => {
  if (isArchivedClass.value && archivedBusinessTabs.has(tab)) {
    activeTab.value = 'settings'
    return
  }
  if (tab === 'members') {
    loadMembers()
  }
  if (tab === 'resources') {
    loadResourceTab()
  }
  if (tab === 'assignments') {
    loadAssignments()
  }
  if (tab === 'analytics') {
    loadClassAnalytics()
  }
})

onMounted(async () => {
  await loadData()
  await Promise.all([
    loadDomains(),
    loadKnowledgeBases(),
  ])
  if (activeTab.value === 'members') {
    await loadMembers()
  }
  if (activeTab.value === 'resources') {
    await loadResourceTab()
  }
  if (activeTab.value === 'assignments') {
    await loadAssignments()
  }
  if (activeTab.value === 'analytics') {
    await loadClassAnalytics()
  }
})
</script>

<style lang="less" scoped>
.exam-page {
  flex: 1;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
}

.exam-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;

  h2 {
    margin: 8px 0 0;
    font-size: 22px;
    line-height: 30px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;

  span {
    display: block;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    margin-bottom: 8px;
  }

  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 15px;
    font-weight: 600;
  }
}

.detail-tabs {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  padding: 0 16px 16px;
}

.tab-panel {
  min-height: 260px;
  padding: 18px 0 0;

  h3 {
    margin: 0 0 6px;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 0 0 16px;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.panel-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.section-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.resource-section {
  margin-top: 18px;
}

.muted-text {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.resource-name-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;

  strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
    font-weight: 600;
  }

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.task-error {
  margin-top: 4px;
  color: var(--td-error-color);
  font-size: 12px;
  line-height: 18px;
}

.assignment-form {
  max-height: calc(100vh - 220px);
  overflow-y: auto;
  padding-right: 4px;
}

.assignment-group-control {
  :deep(.t-form__controls-content) {
    flex-direction: column;
    gap: 8px;
  }
}

.assignment-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.assignment-card {
  display: flex;
  min-height: 148px;
  flex-direction: column;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.assignment-card__header,
.assignment-card__footer {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.assignment-card__header {
  .assignment-card__title {
    min-width: 0;
    flex: 1;
  }

  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 15px;
    line-height: 22px;
    font-weight: 600;
  }

  span {
    display: block;
    margin-top: 4px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.assignment-card__instructions {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--td-text-color-secondary);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: 12px;
  line-height: 18px;
}

.assignment-card__footer {
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-stroke);

  span {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.assignment-card__actions {
  min-width: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.assignment-progress {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.assignment-progress__toolbar {
  display: flex;
  justify-content: flex-end;
  min-width: 0;
}

.assignment-progress__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.analytics-panel {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.analytics-summary-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.analytics-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 16px;
}

.flow-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;

  div {
    padding: 16px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
    font-weight: 600;
    text-align: center;
  }
}

@media (max-width: 768px) {
  .exam-page {
    padding: 20px 16px;
  }

  .summary-grid,
  .flow-grid,
  .assignment-progress__summary,
  .analytics-summary-grid,
  .dialog-grid {
    grid-template-columns: 1fr;
  }

  .panel-title-row,
  .section-title-row {
    flex-direction: column;
  }

  .assignment-card__header,
  .assignment-card__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .assignment-card__actions {
    justify-content: flex-start;
  }

  .assignment-progress__toolbar {
    justify-content: flex-start;
  }
}
</style>
