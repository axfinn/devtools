<template>
  <div class="container mx-auto p-4 max-w-7xl">
    <!-- 未启用提示 -->
    <el-alert
      v-if="statusChecked && !nfsEnabled"
      title="NFS 分享功能未启用"
      type="warning"
      description="请在后端 config.yaml 中配置 nfs_share.enabled: true 及 mount_path 后重启服务。"
      show-icon
      :closable="false"
      class="mb-4"
    />

    <!-- 超管登录区 -->
    <el-card v-if="nfsEnabled && !adminLoggedIn" class="mb-4 max-w-md mx-auto">
      <template #header>
        <div class="flex items-center gap-2">
          <el-icon><Lock /></el-icon>
          <span class="font-bold text-lg">NFS 分享 · 超管登录</span>
        </div>
      </template>
      <el-form @submit.prevent="loginAdmin" class="space-y-4">
        <el-form-item label="超管密码">
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="请输入超管密码"
            show-password
            @keyup.enter="loginAdmin"
          />
        </el-form-item>
        <el-button type="primary" :loading="loginLoading" @click="loginAdmin" class="w-full">
          登录
        </el-button>
      </el-form>
    </el-card>

    <!-- 主面板（已登录） -->
    <template v-if="nfsEnabled && adminLoggedIn">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <el-icon class="text-xl text-blue-500"><FolderOpened /></el-icon>
          <span class="text-xl font-bold">NFS 文件分享</span>
          <el-tag type="success" size="small">已登录</el-tag>
        </div>
        <el-button size="small" @click="logout">退出登录</el-button>
      </div>

      <el-tabs v-model="activeTab" class="nfs-tabs">
        <!-- Tab 1: 文件浏览 & 创建分享 -->
        <el-tab-pane label="文件浏览 & 创建分享" name="browse">
          <div class="browse-layout mt-2">
            <!-- 左侧：目录浏览 -->
            <el-card class="browse-file-card">
              <template #header>
                <div class="flex flex-col gap-1">
                  <div class="flex items-center justify-between">
                    <span class="font-semibold">目录浏览</span>
                  </div>
                  <div class="flex items-center gap-1 text-sm text-gray-500 overflow-hidden">
                    <span class="shrink-0">路径：</span>
                    <el-breadcrumb separator="/" class="nfs-breadcrumb">
                      <el-breadcrumb-item
                        v-for="(seg, i) in mobileBreadcrumbs"
                        :key="i"
                        :class="{ 'cursor-pointer text-blue-500': i < mobileBreadcrumbs.length - 1 }"
                        @click="i < mobileBreadcrumbs.length - 1 && navigateTo(seg.path)"
                      >{{ seg.name }}</el-breadcrumb-item>
                    </el-breadcrumb>
                  </div>
                </div>
              </template>
              <el-skeleton :loading="browseLoading" animated>
                <template #default>
                  <el-table
                    :data="dirEntries"
                    size="small"
                    stripe
                    @row-click="handleEntryClick"
                    style="cursor: pointer;"
                  >
                    <el-table-column width="36">
                      <template #default="{ row }">
                        <el-icon :class="row.is_dir ? 'text-yellow-500' : 'text-blue-400'">
                          <Folder v-if="row.is_dir" />
                          <Document v-else />
                        </el-icon>
                      </template>
                    </el-table-column>
                    <el-table-column label="名称" prop="name" min-width="120">
                      <template #default="{ row }">
                        <div>
                          <span :class="row.is_dir ? 'font-medium text-yellow-700' : ''">{{ row.name }}</span>
                          <div class="text-xs text-gray-400 sm-hidden-info">
                            <span v-if="!row.is_dir">{{ formatSize(row.size) }}</span>
                            <span v-if="!row.is_dir" class="mx-1">·</span>
                            <span>{{ formatDate(row.mod_time) }}</span>
                          </div>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column label="大小" width="90" class-name="col-hide-mobile">
                      <template #default="{ row }">
                        <span v-if="!row.is_dir" class="text-gray-500 text-xs">{{ formatSize(row.size) }}</span>
                        <span v-else class="text-gray-400 text-xs">—</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="修改时间" width="140" class-name="col-hide-mobile">
                      <template #default="{ row }">
                        <span class="text-gray-400 text-xs">{{ formatDate(row.mod_time) }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80">
                      <template #default="{ row }">
                        <el-button
                          v-if="!row.is_dir"
                          size="small"
                          type="primary"
                          link
                          @click.stop="selectFile(row)"
                        >选为分享</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </template>
              </el-skeleton>
            </el-card>

            <!-- 右侧：创建分享 -->
            <el-card class="browse-create-card">
              <template #header>
                <span class="font-semibold">创建分享</span>
              </template>
              <el-form :model="createForm" label-width="90px" size="default">
                <el-form-item label="选中文件">
                  <div v-if="createForm.filePath" class="text-sm">
                    <div class="font-medium truncate" :title="createForm.filePath">
                      {{ createForm.filePath.split('/').pop() }}
                    </div>
                    <div class="text-gray-400 text-xs truncate" :title="createForm.filePath">
                      {{ createForm.filePath }}
                    </div>
                    <div class="text-gray-400 text-xs">{{ formatSize(selectedFileSize) }}</div>
                  </div>
                  <span v-else class="text-gray-400 text-sm">请在左侧点击"选为分享"</span>
                </el-form-item>
                <el-form-item label="显示名称">
                  <el-input v-model="createForm.name" placeholder="分享名称（必填）" />
                </el-form-item>
                <el-form-item label="访问次数">
                  <el-input-number
                    v-model="createForm.maxViews"
                    :min="1"
                    :max="99999"
                    controls-position="right"
                    class="w-full"
                  />
                </el-form-item>
                <el-form-item label="有效期">
                  <el-select v-model="createForm.expiresDays" class="w-full">
                    <el-option :value="0" label="永不过期" />
                    <el-option :value="1" label="1 天" />
                    <el-option :value="3" label="3 天" />
                    <el-option :value="7" label="7 天" />
                    <el-option :value="30" label="30 天" />
                    <el-option :value="90" label="90 天" />
                    <el-option :value="365" label="1 年" />
                  </el-select>
                </el-form-item>
                <el-form-item label="访问密码">
                  <el-input
                    v-model="createForm.password"
                    type="password"
                    placeholder="留空则无需密码"
                    show-password
                  />
                </el-form-item>
                <el-form-item label="录音">
                  <el-switch v-model="createForm.recordEnabled" active-text="开启（访客观看时自动录音）" />
                </el-form-item>
                <el-form-item>
                  <el-button
                    type="primary"
                    :loading="createLoading"
                    :disabled="!createForm.filePath || !createForm.name"
                    class="w-full"
                    @click="createShare"
                  >创建分享链接</el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <!-- Tab 2: 挂载管理 -->
        <el-tab-pane label="挂载管理" name="mounts">
          <div class="mt-2">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm text-gray-500">挂载点来自 config.yaml → nfs_share.mounts</span>
              <el-button size="small" :loading="mountsLoading" @click="loadMounts">
                <el-icon><Refresh /></el-icon> 刷新状态
              </el-button>
            </div>
            <el-empty v-if="!mountsLoading && mountsList.length === 0" description="暂无挂载点，请在 config.yaml 中配置 nfs_share.mounts" />
            <div class="grid gap-3">
              <el-card
                v-for="m in mountsList"
                :key="m.name"
                :class="['border-l-4', m.mounted ? 'border-l-green-400' : 'border-l-red-400']"
                shadow="never"
              >
                <div class="flex items-start justify-between">
                  <div class="space-y-1">
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-base">{{ m.name }}</span>
                      <el-tag :type="m.mounted ? 'success' : 'danger'" size="small">
                        {{ m.mounted ? '已挂载' : '未挂载' }}
                      </el-tag>
                      <el-tag size="small" type="info">{{ m.type.toUpperCase() }}</el-tag>
                    </div>
                    <div class="text-sm text-gray-600">
                      <span v-if="m.type === 'nfs'">{{ m.host }}:{{ m.export }}</span>
                      <span v-else-if="m.type === 'local'">{{ m.export }}</span>
                      <span v-else>//{{ m.host }}/{{ m.share }}
                        <span v-if="m.username" class="text-gray-400 ml-1">（用户：{{ m.username }}）</span>
                      </span>
                    </div>
                    <div class="text-xs text-gray-400">本地挂载点：{{ m.local_path }}</div>
                    <div v-if="m.mounted_at" class="text-xs text-gray-400">
                      挂载时间：{{ formatDate(m.mounted_at) }}
                    </div>
                    <div v-if="m.error" class="text-xs text-red-500 mt-1">
                      错误：{{ m.error }}
                    </div>
                  </div>
                  <div class="flex gap-2 ml-4 flex-shrink-0">
                    <el-button
                      size="small"
                      type="primary"
                      :loading="mountActionLoading === m.name + '_remount'"
                      @click="remount(m.name)"
                    >重新挂载</el-button>
                    <el-button
                      v-if="m.mounted"
                      size="small"
                      type="warning"
                      :loading="mountActionLoading === m.name + '_umount'"
                      @click="umount(m.name)"
                    >卸载</el-button>
                  </div>
                </div>
              </el-card>
            </div>

            <!-- config.yaml 示例 -->
            <el-collapse class="mt-4">
              <el-collapse-item title="config.yaml 配置示例" name="example">
                <pre class="bg-gray-50 rounded p-3 text-xs overflow-x-auto">{{configExample}}</pre>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-tab-pane>

        <!-- Tab: 上传文件 -->
        <el-tab-pane label="上传文件" name="upload">
          <div class="mt-2 max-w-xl mx-auto space-y-4">

            <!-- 目标目录选择 -->
            <el-card shadow="never">
              <div class="flex items-center gap-2">
                <span class="text-sm text-gray-600 shrink-0">上传到：</span>
                <el-tag v-if="uploadTargetDir" type="info" closable @close="uploadTargetDir = ''">
                  {{ uploadTargetDir }}
                </el-tag>
                <span v-else class="text-sm text-gray-400">内置上传目录（默认）</span>
                <el-button size="small" class="ml-auto" @click="openTargetDirPicker">选择目录</el-button>
              </div>
            </el-card>

            <!-- 拖拽/粘贴上传区 -->
            <div
              class="upload-drop-zone"
              :class="{ 'is-dragover': uploadDragover }"
              @dragover.prevent="uploadDragover = true"
              @dragleave="uploadDragover = false"
              @drop.prevent="onUploadDrop"
              @paste.prevent="onUploadPaste"
              @click="$refs.uploadFileInput.click()"
              tabindex="0"
              @keydown.enter="$refs.uploadFileInput.click()"
            >
              <div v-if="!uploadFile" class="text-center text-gray-400 select-none">
                <div class="text-4xl mb-2">📁</div>
                <div class="text-sm">点击选择文件、拖拽文件到此处</div>
                <div class="text-xs mt-1">或 Ctrl+V 粘贴剪贴板图片</div>
              </div>
              <div v-else class="text-center select-none">
                <div class="text-2xl mb-1">📄</div>
                <div class="text-sm font-medium text-gray-700">{{ uploadFile.name }}</div>
                <div class="text-xs text-gray-400 mt-1">{{ formatSize(uploadFile.size) }}</div>
              </div>
              <input ref="uploadFileInput" type="file" class="hidden" @change="onUploadFileChange" />
            </div>

            <!-- 进度 -->
            <div v-if="uploadProgress > 0 || uploadDone">
              <el-progress :percentage="uploadProgress" :status="uploadDone ? 'success' : ''" />
              <div class="text-xs text-gray-400 mt-1 text-center">
                {{ uploadDone ? '上传完成' : `分片 ${uploadChunkDone} / ${uploadChunkTotal}` }}
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex gap-2">
              <el-button
                type="primary"
                :loading="uploading"
                :disabled="!uploadFile || uploadDone"
                class="flex-1"
                @click="startUpload"
              >上传</el-button>
              <el-button :disabled="uploading" @click="resetUpload">重置</el-button>
            </div>

            <!-- 上传完成后的创建分享表单 -->
            <el-card v-if="uploadDone && uploadedFilePath" shadow="never" class="border-green-300">
              <template #header><span class="text-sm font-semibold text-green-700">创建分享链接</span></template>
              <el-form :model="uploadShareForm" label-width="80px" size="small">
                <el-form-item label="文件名">
                  <el-input v-model="uploadShareForm.name" placeholder="分享显示名称" />
                </el-form-item>
                <el-form-item label="访问次数">
                  <el-input-number v-model="uploadShareForm.maxViews" :min="1" :max="9999" controls-position="right" class="w-full" />
                </el-form-item>
                <el-form-item label="有效期">
                  <el-select v-model="uploadShareForm.expiresDays" class="w-full">
                    <el-option :value="1" label="1 天" />
                    <el-option :value="3" label="3 天" />
                    <el-option :value="7" label="7 天" />
                    <el-option :value="30" label="30 天" />
                    <el-option :value="90" label="90 天" />
                    <el-option :value="365" label="1 年" />
                  </el-select>
                </el-form-item>
                <el-form-item label="访问密码">
                  <el-input v-model="uploadShareForm.password" type="password" placeholder="留空则无需密码" show-password />
                </el-form-item>
                <el-form-item>
                  <el-button type="success" :loading="uploadShareLoading" class="w-full" @click="createUploadShare">
                    创建分享链接
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <!-- 目标目录选择弹窗 -->
        <el-dialog v-model="targetDirPickerVisible" title="选择上传目录" :width="dialogWidth">
          <div class="space-y-2">
            <!-- 面包屑 -->
            <div class="flex items-center gap-1 text-sm text-gray-500 flex-wrap">
              <span
                class="cursor-pointer text-blue-500 hover:underline"
                @click="browseTargetDir('.')"
              >根目录</span>
              <template v-for="(seg, i) in targetDirBreadcrumbs" :key="i">
                <span class="text-gray-300">/</span>
                <span
                  :class="i < targetDirBreadcrumbs.length - 1 ? 'cursor-pointer text-blue-500 hover:underline' : 'text-gray-700'"
                  @click="i < targetDirBreadcrumbs.length - 1 && browseTargetDir(seg.path)"
                >{{ seg.name }}</span>
              </template>
            </div>
            <!-- 当前选中 -->
            <div v-if="targetDirCurrent !== '.'" class="text-xs text-green-600 bg-green-50 rounded px-2 py-1">
              将上传到：{{ targetDirCurrent }}
            </div>
            <!-- 目录列表 -->
            <div v-loading="targetDirLoading" class="max-h-64 overflow-y-auto border rounded">
              <div
                v-for="entry in targetDirEntries"
                :key="entry.path"
                class="flex items-center gap-2 px-3 py-2 hover:bg-gray-50 cursor-pointer text-sm border-b last:border-b-0"
                @click="entry.is_dir && browseTargetDir(entry.path)"
              >
                <span>{{ entry.is_dir ? '📁' : '📄' }}</span>
                <span :class="entry.is_dir ? 'text-blue-600' : 'text-gray-400'">{{ entry.name }}</span>
              </div>
              <div v-if="!targetDirLoading && targetDirEntries.length === 0" class="text-center text-gray-400 py-4 text-sm">空目录</div>
            </div>
          </div>
          <template #footer>
            <el-button @click="targetDirPickerVisible = false">取消</el-button>
            <el-button type="primary" @click="confirmTargetDir">
              {{ targetDirCurrent === '.' ? '使用默认目录' : `选择 ${targetDirCurrent}` }}
            </el-button>
          </template>
        </el-dialog>

        <!-- Tab 3: 分享列表 -->
        <el-tab-pane name="list">
          <template #label>
            <span>分享列表</span>
            <el-badge v-if="listTotal > 0" :value="listTotal" class="ml-1" type="info" />
          </template>
          <div class="mt-2">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm text-gray-500">共 {{ listTotal }} 条分享记录</span>
              <el-button size="small" :loading="listLoading" @click="loadShareList">
                <el-icon><Refresh /></el-icon> 刷新
              </el-button>
            </div>
            <el-table :data="shareList" size="small" stripe v-loading="listLoading">
              <el-table-column label="名称" min-width="140">
                <template #default="{ row }">
                  <div class="font-medium">{{ row.name }}</div>
                  <div class="text-xs text-gray-400 truncate" :title="row.file_path">{{ row.file_path }}</div>
                  <!-- 移动端内联显示额外信息 -->
                  <div class="sm-hidden-info text-xs text-gray-400 mt-0.5 flex flex-wrap gap-x-2">
                    <span>{{ formatSize(row.file_size) }}</span>
                    <el-tag :type="getShareStatus(row).type" size="small" class="!text-xs">{{ getShareStatus(row).label }}</el-tag>
                    <span :class="row.expires_at && isExpired(row.expires_at) ? 'text-red-500' : ''">
                      {{ row.expires_at ? formatDate(row.expires_at) : '永不过期' }}
                    </span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="大小" width="80" class-name="col-hide-mobile">
                <template #default="{ row }">
                  <span class="text-xs">{{ formatSize(row.file_size) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" width="110" align="center">
                <template #default="{ row }">
                  <el-progress
                    :percentage="row.max_views > 0 ? Math.round((row.views / row.max_views) * 100) : 0"
                    :status="row.views >= row.max_views ? 'exception' : ''"
                    :stroke-width="6"
                  />
                  <div class="text-xs text-center mt-1">{{ row.views }} / {{ row.max_views }}</div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="72" align="center" class-name="col-hide-mobile">
                <template #default="{ row }">
                  <el-tag :type="getShareStatus(row).type" size="small">{{ getShareStatus(row).label }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="有效期" width="120" class-name="col-hide-mobile">
                <template #default="{ row }">
                  <span v-if="row.expires_at" class="text-xs" :class="isExpired(row.expires_at) ? 'text-red-500' : 'text-gray-500'">
                    {{ formatDate(row.expires_at) }}
                  </span>
                  <span v-else class="text-xs text-green-500">永不过期</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="160" align="center">
                <template #default="{ row }">
                  <el-button size="small" type="primary" link @click="copyShareLink(row)">复制</el-button>
                  <el-button size="small" type="info" link @click="viewLogs(row)">日志</el-button>
                  <el-button size="small" link @click="openEditDialog(row)">调整</el-button>
                  <el-button size="small" type="danger" link @click="deleteShare(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="flex justify-end mt-3">
              <el-pagination
                v-model:current-page="listPage"
                :page-size="listPageSize"
                :total="listTotal"
                layout="prev, pager, next"
                @current-change="loadShareList"
              />
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- 访问日志弹窗 -->
    <el-dialog
      v-model="logsDialogVisible"
      :title="`访问日志 · ${currentShare?.name || ''}`"
      :width="dialogWidth"
      destroy-on-close
    >
      <div class="flex items-center justify-between mb-3">
        <span class="text-sm text-gray-500">共 {{ logsTotal }} 条访问记录</span>
        <el-button size="small" @click="loadLogs">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <el-table :data="logList" size="small" stripe v-loading="logsLoading" max-height="400">
        <el-table-column label="时间" min-width="120">
          <template #default="{ row }">
            <span class="text-xs">{{ formatDate(row.accessed_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="IP" width="120" prop="client_ip" class-name="col-hide-mobile" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="getLogStatusTag(row.status)" size="small">
              {{ getLogStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="传输大小" width="90" class-name="col-hide-mobile">
          <template #default="{ row }">
            <span class="text-xs">{{ row.bytes_sent > 0 ? formatSize(row.bytes_sent) : '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="User-Agent" min-width="160" class-name="col-hide-mobile">
          <template #default="{ row }">
            <span class="text-xs text-gray-500 truncate block" :title="row.user_agent">{{ row.user_agent }}</span>
          </template>
        </el-table-column>
        <el-table-column label="录音" width="100" align="center">
          <template #default="{ row }">
            <template v-if="row.audio_url">
              <el-button
                v-for="(url, idx) in parseAudioUrls(row.audio_url)"
                :key="idx"
                size="small"
                type="success"
                link
                @click="playRecord(url)"
              >▶ {{ idx + 1 }}</el-button>
            </template>
            <span v-else class="text-xs text-gray-400">—</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="flex justify-end mt-3">
        <el-pagination
          v-model:current-page="logsPage"
          :page-size="logsPageSize"
          :total="logsTotal"
          layout="prev, pager, next"
          @current-change="loadLogs"
        />
      </div>
    </el-dialog>

    <!-- 录音播放弹窗 -->
    <el-dialog v-model="recordDialogVisible" title="录音播放" width="420px" destroy-on-close>
      <audio v-if="recordPlayURL" :src="recordPlayURL" controls autoplay style="width:100%" />
    </el-dialog>

    <!-- 调整分享配置弹窗 -->
    <el-dialog v-model="editDialogVisible" title="调整分享配置" :width="dialogWidth">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="当前次数">
          <span class="text-sm text-gray-600">已访问 {{ editTarget?.views }} / {{ editTarget?.max_views }} 次</span>
        </el-form-item>
        <el-form-item label="追加次数">
          <el-input-number v-model="editForm.addViews" :min="0" :max="9999" controls-position="right" class="w-full" />
          <div class="text-xs text-gray-400 mt-1">在现有最大次数基础上增加</div>
        </el-form-item>
        <el-form-item label="延期天数">
          <el-input-number v-model="editForm.addDays" :min="0" :max="3650" controls-position="right" class="w-full" />
          <div class="text-xs text-gray-400 mt-1">在现有到期日基础上延期</div>
        </el-form-item>
        <el-form-item label="一起看">
          <el-switch v-model="editForm.watchEnabled" active-text="开启" inactive-text="关闭" />
          <div class="text-xs text-gray-400 mt-1">开启后访客进入链接自动加入一起看</div>
        </el-form-item>
        <el-form-item label="录音">
          <el-switch v-model="editForm.recordEnabled" active-text="开启" inactive-text="关闭" />
          <div class="text-xs text-gray-400 mt-1">开启后访客观看时自动录音并保存</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">确认更新</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Lock, FolderOpened, Folder, Document, Refresh
} from '@element-plus/icons-vue'

// -------- 状态 --------
const statusChecked = ref(false)
const nfsEnabled = ref(false)
const adminLoggedIn = ref(false)
const adminPassword = ref('')
const loginLoading = ref(false)
const activeTab = ref('browse')

// 目录浏览
const currentPath = ref('.')
const dirEntries = ref([])
const browseLoading = ref(false)

// 创建分享
const selectedFileSize = ref(0)
const createForm = reactive({
  filePath: '',
  name: '',
  maxViews: 5,
  expiresDays: 7,
  password: '',
  recordEnabled: false
})
const createLoading = ref(false)

// 分享列表
const shareList = ref([])
const listPage = ref(1)
const listPageSize = 20
const listTotal = ref(0)
const listLoading = ref(false)

// 访问日志弹窗
const logsDialogVisible = ref(false)
const currentShare = ref(null)
const logList = ref([])
const logsPage = ref(1)
const logsPageSize = 50
const logsTotal = ref(0)
const logsLoading = ref(false)

// 挂载管理
const mountsList = ref([])
const mountsLoading = ref(false)
const mountActionLoading = ref('')

// 上传文件
const uploadFile = ref(null)
const uploadDragover = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadChunkDone = ref(0)
const uploadChunkTotal = ref(0)
const uploadDone = ref(false)
const uploadedFilePath = ref('')
const uploadShareLoading = ref(false)
const uploadShareForm = reactive({ name: '', maxViews: 5, expiresDays: 7, password: '' })
const uploadTargetDir = ref('') // 目标目录，空=默认

// 目标目录选择弹窗
const targetDirPickerVisible = ref(false)
const targetDirCurrent = ref('.')
const targetDirEntries = ref([])
const targetDirLoading = ref(false)
const targetDirBreadcrumbs = computed(() => {
  if (!targetDirCurrent.value || targetDirCurrent.value === '.') return []
  const segs = []
  const parts = targetDirCurrent.value.split('/')
  let built = ''
  for (const p of parts) {
    built = built ? `${built}/${p}` : p
    segs.push({ name: p, path: built })
  }
  return segs
})

// 编辑弹窗
const editDialogVisible = ref(false)
const editTarget = ref(null)
const editForm = reactive({ addViews: 0, addDays: 0, watchEnabled: false, recordEnabled: false })
const editLoading = ref(false)

// 录音播放
const recordDialogVisible = ref(false)
const recordPlayURL = ref('')

// -------- 计算属性 --------
const breadcrumbs = computed(() => {
  const segs = [{ name: 'NFS 根目录', path: '.' }]
  if (currentPath.value && currentPath.value !== '.') {
    const parts = currentPath.value.split('/')
    let built = ''
    for (const p of parts) {
      built = built ? `${built}/${p}` : p
      segs.push({ name: p, path: built })
    }
  }
  return segs
})

// 移动端只显示最后2段面包屑
const mobileBreadcrumbs = computed(() => {
  const all = breadcrumbs.value
  if (all.length <= 2) return all
  return [{ name: '…', path: all[all.length - 2].path }, all[all.length - 1]]
})

const dialogWidth = computed(() => window.innerWidth < 640 ? '95%' : '560px')

// -------- 生命周期 --------
onMounted(async () => {
  await checkStatus()
  if (nfsEnabled.value) {
    const saved = sessionStorage.getItem('nfs_admin_password')
    if (saved) {
      adminPassword.value = saved
      await loginAdmin(true)
    }
  }
})

// -------- 状态检查 --------
async function checkStatus() {
  try {
    const res = await fetch('/api/nfsshare/status')
    const data = await res.json()
    nfsEnabled.value = !!data.enabled
  } catch {
    nfsEnabled.value = false
  } finally {
    statusChecked.value = true
  }
}

// -------- 超管登录 --------
async function loginAdmin(silent = false) {
  if (!adminPassword.value) {
    if (!silent) ElMessage.warning('请输入超管密码')
    return
  }
  loginLoading.value = true
  try {
    // 用浏览目录来验证密码
    const res = await fetch(`/api/nfsshare/admin/browse?path=.&admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (res.ok) {
      adminLoggedIn.value = true
      sessionStorage.setItem('nfs_admin_password', adminPassword.value)
      await Promise.all([loadDir('.'), loadShareList(), loadMounts()])
    } else {
      const data = await res.json()
      if (!silent) ElMessage.error(data.error || '密码错误')
      adminPassword.value = ''
      sessionStorage.removeItem('nfs_admin_password')
    }
  } catch {
    if (!silent) ElMessage.error('网络错误，请重试')
  } finally {
    loginLoading.value = false
  }
}

function logout() {
  adminLoggedIn.value = false
  adminPassword.value = ''
  sessionStorage.removeItem('nfs_admin_password')
  dirEntries.value = []
  shareList.value = []
  currentPath.value = '.'
}

// -------- 目录浏览 --------
async function loadDir(path) {
  browseLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/browse?path=${encodeURIComponent(path)}&admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '无法读取目录')
      return
    }
    const data = await res.json()
    currentPath.value = path
    dirEntries.value = (data.entries || []).sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
  } catch {
    ElMessage.error('网络错误')
  } finally {
    browseLoading.value = false
  }
}

function handleEntryClick(row) {
  if (row.is_dir) {
    loadDir(row.path)
  }
}

function navigateTo(path) {
  loadDir(path)
}

function selectFile(row) {
  createForm.filePath = row.path
  createForm.name = row.name
  selectedFileSize.value = row.size
  ElMessage.success(`已选择：${row.name}`)
}

// -------- 创建分享 --------
async function createShare() {
  if (!createForm.filePath || !createForm.name) {
    ElMessage.warning('请先选择文件并填写名称')
    return
  }
  createLoading.value = true
  try {
    const res = await fetch('/api/nfsshare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword.value,
        name: createForm.name,
        file_path: createForm.filePath,
        max_views: createForm.maxViews,
        expires_days: createForm.expiresDays,
        password: createForm.password || '',
        record_enabled: createForm.recordEnabled || false
      })
    })
    const data = await res.json()
    if (!res.ok) {
      ElMessage.error(data.error || '创建失败')
      return
    }
    const link = `${location.origin}/nfs/${data.id}`
    await navigator.clipboard.writeText(link).catch(() => {})
    ElMessage.success(`创建成功！链接已复制：${link}`)
    createForm.filePath = ''
    createForm.name = ''
    createForm.maxViews = 5
    createForm.expiresDays = 7
    selectedFileSize.value = 0
    // 切换到列表 tab 并刷新
    activeTab.value = 'list'
    await loadShareList()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    createLoading.value = false
  }
}

// -------- 挂载管理 --------
async function loadMounts() {
  mountsLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/mounts?admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (!res.ok) return
    const data = await res.json()
    mountsList.value = data.mounts || []
  } catch {
    // ignore
  } finally {
    mountsLoading.value = false
  }
}

async function remount(name) {
  mountActionLoading.value = name + '_remount'
  try {
    const res = await fetch(
      `/api/nfsshare/admin/mounts/${name}/remount?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'POST' }
    )
    const data = await res.json()
    if (res.ok) {
      ElMessage.success(`${name} 挂载成功`)
    } else {
      ElMessage.error(data.error || '挂载失败')
    }
    await loadMounts()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    mountActionLoading.value = ''
  }
}

async function umount(name) {
  try {
    await ElMessageBox.confirm(`确定卸载 ${name}？`, '确认卸载', { type: 'warning' })
  } catch { return }
  mountActionLoading.value = name + '_umount'
  try {
    const res = await fetch(
      `/api/nfsshare/admin/mounts/${name}/umount?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'POST' }
    )
    const data = await res.json()
    if (res.ok) {
      ElMessage.success(`${name} 已卸载`)
    } else {
      ElMessage.error(data.error || '卸载失败')
    }
    await loadMounts()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    mountActionLoading.value = ''
  }
}

const configExample = `nfs_share:
  enabled: true
  admin_password: "your_super_admin_pass"
  max_file_size_mb: 0   # 0 = 不限制

  mounts:
    # NFS 示例（无需密码，服务端 /etc/exports 控制权限）
    - name: "project-files"
      type: nfs
      host: "192.168.1.100"
      export: "/exports/data"
      options: "soft,timeo=30"   # 可选

    # SMB/CIFS 示例（Windows 共享 / Samba）
    - name: "team-share"
      type: smb
      host: "192.168.1.200"
      share: "TeamDocs"
      username: "alice"
      password: "secret"
      domain: "CORP"     # 可选，域账号时填写

    # 本地目录示例（直接映射，不执行 mount）
    - name: "local-backup"
      type: local
      export: "/mnt/backup"`

// -------- 分享列表 --------
async function loadShareList() {
  listLoading.value = true
  try {
    const res = await fetch(
      `/api/nfsshare/admin/list?admin_password=${encodeURIComponent(adminPassword.value)}&page=${listPage.value}&page_size=${listPageSize}`
    )
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '获取列表失败')
      return
    }
    const data = await res.json()
    shareList.value = data.shares || []
    listTotal.value = data.total || 0
  } catch {
    ElMessage.error('网络错误')
  } finally {
    listLoading.value = false
  }
}

function copyShareLink(row) {
  const link = `${location.origin}/nfs/${row.id}`
  navigator.clipboard.writeText(link)
    .then(() => ElMessage.success('链接已复制'))
    .catch(() => ElMessage.info(`链接：${link}`))
}

async function deleteShare(row) {
  try {
    await ElMessageBox.confirm(`确定删除分享「${row.name}」及其所有访问记录？`, '确认删除', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    const res = await fetch(
      `/api/nfsshare/admin/${row.id}?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'DELETE' }
    )
    if (res.ok) {
      ElMessage.success('删除成功')
      await loadShareList()
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

// -------- 访问日志 --------
function viewLogs(row) {
  currentShare.value = row
  logsPage.value = 1
  logsDialogVisible.value = true
  loadLogs()
}

async function loadLogs() {
  if (!currentShare.value) return
  logsLoading.value = true
  try {
    const res = await fetch(
      `/api/nfsshare/admin/${currentShare.value.id}/logs?admin_password=${encodeURIComponent(adminPassword.value)}&page=${logsPage.value}&page_size=${logsPageSize}`
    )
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '获取日志失败')
      return
    }
    const data = await res.json()
    logList.value = data.logs || []
    logsTotal.value = data.total || 0
  } catch {
    ElMessage.error('网络错误')
  } finally {
    logsLoading.value = false
  }
}

function parseAudioUrls(audioUrl) {
  if (!audioUrl) return []
  try {
    const arr = JSON.parse(audioUrl)
    if (Array.isArray(arr)) return arr
  } catch {}
  return [audioUrl]
}

function playRecord(url) {
  recordPlayURL.value = `${url}?admin_password=${encodeURIComponent(adminPassword.value)}`
  recordDialogVisible.value = true
}

// -------- 上传文件 --------
const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB

function onUploadFileChange(e) {
  const f = e.target.files[0]
  if (f) setUploadFile(f)
}
function onUploadDrop(e) {
  uploadDragover.value = false
  const f = e.dataTransfer.files[0]
  if (f) setUploadFile(f)
}
function onUploadPaste(e) {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.kind === 'file') {
      const f = item.getAsFile()
      if (f) { setUploadFile(f); break }
    }
  }
}
function setUploadFile(f) {
  uploadFile.value = f
  uploadDone.value = false
  uploadProgress.value = 0
  uploadChunkDone.value = 0
  uploadedFilePath.value = ''
  uploadShareForm.name = f.name
}
function resetUpload() {
  uploadFile.value = null
  uploadDone.value = false
  uploadProgress.value = 0
  uploadChunkDone.value = 0
  uploadedFilePath.value = ''
}

async function openTargetDirPicker() {
  targetDirPickerVisible.value = true
  targetDirCurrent.value = '.'
  await browseTargetDir('.')
}
async function browseTargetDir(path) {
  targetDirCurrent.value = path
  targetDirLoading.value = true
  targetDirEntries.value = []
  try {
    const res = await fetch(`/api/nfsshare/admin/browse?admin_password=${encodeURIComponent(adminPassword.value)}&path=${encodeURIComponent(path)}`)
    const data = await res.json()
    // 只显示目录（根路径时全部是挂载点，都是目录）
    targetDirEntries.value = (data.entries || []).filter(e => e.is_dir)
  } catch {
    ElMessage.error('加载目录失败')
  } finally {
    targetDirLoading.value = false
  }
}
function confirmTargetDir() {
  uploadTargetDir.value = targetDirCurrent.value === '.' ? '' : targetDirCurrent.value
  targetDirPickerVisible.value = false
}

async function startUpload() {
  if (!uploadFile.value) return
  uploading.value = true
  uploadProgress.value = 0
  uploadDone.value = false
  const file = uploadFile.value
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1
  uploadChunkTotal.value = totalChunks
  uploadChunkDone.value = 0
  try {
    // 1. init
    const initRes = await fetch('/api/nfsshare/admin/upload/init', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword.value,
        filename: file.name,
        total_size: file.size,
        total_chunks: totalChunks,
        target_dir: uploadTargetDir.value || ''
      })
    })
    const initData = await initRes.json()
    if (!initRes.ok) { ElMessage.error(initData.error || '初始化失败'); return }
    const token = initData.token

    // 2. chunks
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE
      const blob = file.slice(start, start + CHUNK_SIZE)
      const form = new FormData()
      form.append('chunk_index', String(i))
      form.append('chunk', blob, file.name)
      const chunkRes = await fetch(`/api/nfsshare/admin/upload/${token}/chunk`, {
        method: 'POST',
        body: form
      })
      if (!chunkRes.ok) {
        const d = await chunkRes.json()
        ElMessage.error(d.error || `分片 ${i} 上传失败`)
        return
      }
      uploadChunkDone.value = i + 1
      uploadProgress.value = Math.round(((i + 1) / totalChunks) * 100)
    }

    // 3. complete
    const completeRes = await fetch(`/api/nfsshare/admin/upload/${token}/complete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword.value, total_chunks: totalChunks })
    })
    const completeData = await completeRes.json()
    if (!completeRes.ok) { ElMessage.error(completeData.error || '合并失败'); return }
    uploadedFilePath.value = completeData.file_path
    uploadShareForm.name = completeData.filename
    uploadDone.value = true
    uploadProgress.value = 100
    ElMessage.success('上传完成')
  } catch (e) {
    ElMessage.error('上传出错: ' + e.message)
  } finally {
    uploading.value = false
  }
}
async function createUploadShare() {
  if (!uploadedFilePath.value) return
  uploadShareLoading.value = true
  try {
    const body = {
      admin_password: adminPassword.value,
      name: uploadShareForm.name || uploadedFilePath.value.split('/').pop(),
      file_path: uploadedFilePath.value,
      max_views: uploadShareForm.maxViews,
      expires_days: uploadShareForm.expiresDays,
    }
    if (uploadShareForm.password) body.password = uploadShareForm.password
    const res = await fetch('/api/nfsshare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await res.json()
    if (!res.ok) { ElMessage.error(data.error || '创建失败'); return }
    const link = `${location.origin}/nfsshare/${data.id}`
    await navigator.clipboard.writeText(link).catch(() => {})
    ElMessage.success('分享链接已复制到剪贴板')
    resetUpload()
    activeTab.value = 'list'
    await loadShareList()
  } catch (e) {
    ElMessage.error('创建失败: ' + e.message)
  } finally {
    uploadShareLoading.value = false
  }
}

// -------- 编辑分享 --------
function openEditDialog(row) {
  editTarget.value = row
  editForm.addViews = 0
  editForm.addDays = 0
  editForm.watchEnabled = row.watch_enabled || false
  editForm.recordEnabled = row.record_enabled || false
  editDialogVisible.value = true
}

async function submitEdit() {
  if (!editTarget.value) return
  editLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/${editTarget.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword.value,
        add_views: editForm.addViews,
        add_days: editForm.addDays,
        watch_enabled: editForm.watchEnabled,
        record_enabled: editForm.recordEnabled
      })
    })
    const data = await res.json()
    if (!res.ok) {
      ElMessage.error(data.error || '更新失败')
      return
    }
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    await loadShareList()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    editLoading.value = false
  }
}

// -------- 工具函数 --------
function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function isExpired(dateStr) {
  return dateStr && new Date(dateStr) < new Date()
}

function getShareStatus(row) {
  if (row.expires_at && isExpired(row.expires_at)) {
    return { type: 'danger', label: '已过期' }
  }
  if (row.max_views > 0 && row.views >= row.max_views) {
    return { type: 'warning', label: '已耗尽' }
  }
  return { type: 'success', label: '有效' }
}

function getLogStatusTag(status) {
  const map = {
    success: 'success',
    denied_views: 'warning',
    denied_expired: 'warning',
    file_missing: 'danger',
    not_found: 'info',
    error: 'danger'
  }
  return map[status] || 'info'
}

function getLogStatusLabel(status) {
  const map = {
    success: '成功',
    denied_views: '次数耗尽',
    denied_expired: '已过期',
    file_missing: '文件缺失',
    not_found: '不存在',
    error: '错误'
  }
  return map[status] || status
}
</script>

<style scoped>
.nfs-tabs :deep(.el-tabs__content) {
  overflow: visible;
}

.upload-drop-zone {
  border: 2px dashed #d1d5db;
  border-radius: 8px;
  padding: 40px 20px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
  outline: none;
}
.upload-drop-zone:hover,
.upload-drop-zone:focus {
  border-color: #409eff;
  background: #f0f7ff;
}
.upload-drop-zone.is-dragover {
  border-color: #409eff;
  background: #e6f0ff;
}

/* 文件浏览布局：桌面左右，移动端上下 */
.browse-layout {
  display: flex;
  gap: 16px;
  min-height: 400px;
}
.browse-file-card {
  flex: 1;
  min-width: 0;
}
.browse-create-card {
  width: 340px;
  flex-shrink: 0;
}

/* 面包屑不换行截断 */
.nfs-breadcrumb {
  overflow: hidden;
  white-space: nowrap;
}
.nfs-breadcrumb :deep(.el-breadcrumb__item) {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 移动端内联信息默认隐藏（桌面端不显示） */
.sm-hidden-info {
  display: none;
}

/* 移动端响应式 */
@media (max-width: 640px) {
  .browse-layout {
    flex-direction: column;
  }
  .browse-create-card {
    width: 100%;
  }

  /* 移动端隐藏部分列 */
  :deep(.col-hide-mobile) {
    display: none;
  }

  /* 移动端在名称列内联显示额外信息 */
  .sm-hidden-info {
    display: flex;
  }

  /* 分页简化 */
  :deep(.el-pagination .el-pager) {
    display: none;
  }
}
</style>
