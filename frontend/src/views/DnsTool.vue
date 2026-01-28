<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>IP / DNS 查询工具</h2>
    </div>

    <!-- 当前 IP 信息 -->
    <div class="ip-section">
      <div class="section-header">
        <h3>当前 IP 地址</h3>
        <el-button type="primary" size="small" @click="fetchMyIP" :loading="ipLoading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <div class="ip-display" v-if="myIP">
        <div class="ip-value">{{ myIP }}</div>
        <el-button type="primary" link @click="copyToClipboard(myIP)">
          <el-icon><CopyDocument /></el-icon>
          复制
        </el-button>
      </div>
      <div v-else-if="ipLoading" class="ip-loading">
        <el-icon class="is-loading"><Loading /></el-icon>
        获取中...
      </div>
      <div v-else-if="ipError" class="ip-error">
        <el-alert :title="ipError" type="error" show-icon :closable="false" />
      </div>
    </div>

    <!-- DNS 查询 -->
    <div class="dns-section">
      <div class="section-header">
        <h3>DNS 查询 (nslookup)</h3>
      </div>
      <div class="dns-input-row">
        <el-input
          v-model="domain"
          placeholder="输入域名，如: example.com"
          class="domain-input"
          @keyup.enter="lookupDNS"
          clearable
        >
          <template #prepend>域名</template>
        </el-input>
        <el-select v-model="recordType" style="width: 140px">
          <el-option label="全部" value="ALL" />
          <el-option label="A (IPv4)" value="A" />
          <el-option label="AAAA (IPv6)" value="AAAA" />
          <el-option label="CNAME" value="CNAME" />
          <el-option label="MX (邮件)" value="MX" />
          <el-option label="NS (域名服务器)" value="NS" />
          <el-option label="TXT" value="TXT" />
        </el-select>
        <el-button type="primary" @click="lookupDNS" :loading="dnsLoading">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
      </div>

      <div v-if="dnsError" class="dns-error">
        <el-alert :title="dnsError" type="error" show-icon :closable="false" />
      </div>

      <div v-if="dnsResults && Object.keys(dnsResults).length > 0" class="dns-results">
        <!-- A 记录 -->
        <div v-if="dnsResults.a && dnsResults.a.length > 0" class="record-group">
          <div class="record-type">A 记录 (IPv4)</div>
          <div class="record-list">
            <div v-for="(ip, idx) in dnsResults.a" :key="'a-'+idx" class="record-item">
              <span class="record-value">{{ ip }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(ip)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- AAAA 记录 -->
        <div v-if="dnsResults.aaaa && dnsResults.aaaa.length > 0" class="record-group">
          <div class="record-type">AAAA 记录 (IPv6)</div>
          <div class="record-list">
            <div v-for="(ip, idx) in dnsResults.aaaa" :key="'aaaa-'+idx" class="record-item">
              <span class="record-value">{{ ip }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(ip)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- CNAME 记录 -->
        <div v-if="dnsResults.cname && dnsResults.cname.length > 0" class="record-group">
          <div class="record-type">CNAME 记录</div>
          <div class="record-list">
            <div v-for="(name, idx) in dnsResults.cname" :key="'cname-'+idx" class="record-item">
              <span class="record-value">{{ name }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(name)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- MX 记录 -->
        <div v-if="dnsResults.mx && dnsResults.mx.length > 0" class="record-group">
          <div class="record-type">MX 记录 (邮件服务器)</div>
          <div class="record-list">
            <div v-for="(mx, idx) in dnsResults.mx" :key="'mx-'+idx" class="record-item">
              <span class="record-priority">优先级: {{ mx.priority }}</span>
              <span class="record-value">{{ mx.host }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(mx.host)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- NS 记录 -->
        <div v-if="dnsResults.ns && dnsResults.ns.length > 0" class="record-group">
          <div class="record-type">NS 记录 (域名服务器)</div>
          <div class="record-list">
            <div v-for="(ns, idx) in dnsResults.ns" :key="'ns-'+idx" class="record-item">
              <span class="record-value">{{ ns }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(ns)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- TXT 记录 -->
        <div v-if="dnsResults.txt && dnsResults.txt.length > 0" class="record-group">
          <div class="record-type">TXT 记录</div>
          <div class="record-list">
            <div v-for="(txt, idx) in dnsResults.txt" :key="'txt-'+idx" class="record-item txt-item">
              <span class="record-value">{{ txt }}</span>
              <el-button type="primary" link size="small" @click="copyToClipboard(txt)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <!-- 无记录 -->
        <div v-if="isEmptyResult" class="no-records">
          <el-empty description="未找到 DNS 记录" />
        </div>
      </div>
    </div>

    <!-- 常用域名 -->
    <div class="common-domains">
      <h4>快速查询</h4>
      <div class="domain-tags">
        <el-tag
          v-for="d in commonDomains"
          :key="d"
          class="domain-tag"
          @click="quickLookup(d)"
        >
          {{ d }}
        </el-tag>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, CopyDocument, Search, Loading } from '@element-plus/icons-vue'

const myIP = ref('')
const ipLoading = ref(false)
const ipError = ref('')

const domain = ref('')
const recordType = ref('ALL')
const dnsResults = ref(null)
const dnsLoading = ref(false)
const dnsError = ref('')

const commonDomains = ['google.com', 'github.com', 'cloudflare.com', 'baidu.com', 'qq.com']

const isEmptyResult = computed(() => {
  if (!dnsResults.value) return false
  const r = dnsResults.value
  return (!r.a || r.a.length === 0) &&
         (!r.aaaa || r.aaaa.length === 0) &&
         (!r.cname || r.cname.length === 0) &&
         (!r.mx || r.mx.length === 0) &&
         (!r.ns || r.ns.length === 0) &&
         (!r.txt || r.txt.length === 0)
})

const fetchMyIP = async () => {
  ipLoading.value = true
  ipError.value = ''
  myIP.value = ''

  try {
    const response = await fetch('/api/ip')
    if (!response.ok) throw new Error('获取 IP 失败')
    const data = await response.json()
    myIP.value = data.ip
  } catch (err) {
    ipError.value = err.message || '获取 IP 失败'
  } finally {
    ipLoading.value = false
  }
}

const lookupDNS = async () => {
  if (!domain.value.trim()) {
    ElMessage.warning('请输入域名')
    return
  }

  dnsLoading.value = true
  dnsError.value = ''
  dnsResults.value = null

  try {
    const params = new URLSearchParams({
      domain: domain.value.trim(),
      type: recordType.value
    })
    const response = await fetch(`/api/dns?${params}`)
    if (!response.ok) {
      const err = await response.json()
      throw new Error(err.error || 'DNS 查询失败')
    }
    dnsResults.value = await response.json()
  } catch (err) {
    dnsError.value = err.message || 'DNS 查询失败'
  } finally {
    dnsLoading.value = false
  }
}

const quickLookup = (d) => {
  domain.value = d
  lookupDNS()
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  fetchMyIP()
})
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tool-header h2 {
  margin: 0;
  color: #333;
}

:global(.dark) .tool-header h2 {
  color: #e0e0e0;
}

.ip-section,
.dns-section,
.common-domains {
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  padding: 20px;
  border-radius: 8px;
}

:global(.dark) .ip-section,
:global(.dark) .dns-section,
:global(.dark) .common-domains {
  background-color: #1e1e1e;
  border-color: #333;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.section-header h3 {
  margin: 0;
  color: #333;
  font-size: 16px;
}

:global(.dark) .section-header h3 {
  color: #e0e0e0;
}

.ip-display {
  display: flex;
  align-items: center;
  gap: 15px;
}

.ip-value {
  font-size: 28px;
  font-weight: 600;
  color: #4caf50;
  font-family: 'Consolas', 'Monaco', monospace;
}

.ip-loading {
  color: #666;
  display: flex;
  align-items: center;
  gap: 8px;
}

:global(.dark) .ip-loading {
  color: #a0a0a0;
}

.dns-input-row {
  display: flex;
  gap: 10px;
  margin-bottom: 15px;
}

.domain-input {
  flex: 1;
}

.dns-results {
  display: flex;
  flex-direction: column;
  gap: 15px;
  margin-top: 15px;
}

.record-group {
  background-color: #f5f5f5;
  border: 1px solid #e0e0e0;
  padding: 15px;
  border-radius: 6px;
}

:global(.dark) .record-group {
  background-color: #2d2d2d;
  border-color: #404040;
}

.record-type {
  color: #4caf50;
  font-weight: 600;
  margin-bottom: 10px;
  font-size: 14px;
}

.record-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.record-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
}

:global(.dark) .record-item {
  background-color: #1e1e1e;
  border-color: #333;
}

.record-item.txt-item {
  flex-wrap: wrap;
}

.record-item.txt-item .record-value {
  word-break: break-all;
}

.record-value {
  font-family: 'Consolas', 'Monaco', monospace;
  color: #333;
  flex: 1;
}

:global(.dark) .record-value {
  color: #d4d4d4;
}

.record-priority {
  color: #ff9800;
  font-size: 12px;
  padding: 2px 8px;
  background-color: rgba(255, 152, 0, 0.2);
  border-radius: 4px;
}

.common-domains h4 {
  margin: 0 0 15px 0;
  color: #333;
}

:global(.dark) .common-domains h4 {
  color: #e0e0e0;
}

.domain-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.domain-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.domain-tag:hover {
  transform: translateY(-2px);
}

.no-records {
  padding: 20px;
}

@media (max-width: 768px) {
  .dns-input-row {
    flex-direction: column;
  }

  .dns-input-row .el-select {
    width: 100% !important;
  }

  .ip-value {
    font-size: 22px;
  }
}
</style>
