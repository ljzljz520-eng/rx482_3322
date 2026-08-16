const customerList = document.querySelector("#customer-list");
const logList = document.querySelector("#log-list");
const formMessage = document.querySelector("#form-message");
const previewResult = document.querySelector("#preview-result");
const importInput = document.querySelector("#import-input");
const commitButton = document.querySelector("#commit-import");
let previewRecords = null;

const request = async (url, options = {}) => {
  const response = await fetch(url, { headers: { "Content-Type": "application/json" }, ...options });
  const payload = await response.json();
  if (!response.ok) throw new Error(payload.error || "请求失败");
  return payload;
};

const renderCustomers = (customers) => {
  if (!customers.length) {
    customerList.innerHTML = '<p class="empty">暂无客户档案</p>';
    return;
  }
  customerList.innerHTML = `<table><thead><tr><th>客户</th><th>患者 / 关系</th><th>手机号</th><th>城市</th><th>回访</th><th>备注</th></tr></thead><tbody>${customers.map((c) => `<tr><td><strong>${c.name}</strong><small>${c.id}</small></td><td>${c.patient_name}<small>${c.relationship}</small></td><td>${c.phone}</td><td>${c.service_city}</td><td>${c.follow_up_at || "未安排"}</td><td>${c.note || "-"}</td></tr>`).join("")}</tbody></table>`;
};

const renderLogs = (logs) => {
  logList.innerHTML = logs.length ? logs.map((entry) => `<li><span>${entry.action}</span><strong>${entry.detail}</strong><small>${entry.at}</small></li>`).join("") : '<li class="empty">暂无操作</li>';
};

const refresh = async () => {
  renderCustomers(await request("/api/customers"));
  renderLogs(await request("/api/operation-logs"));
};

document.querySelector("#customer-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(event.currentTarget));
  try {
    await request("/api/customers", { method: "POST", body: JSON.stringify(data) });
    event.currentTarget.reset();
    formMessage.textContent = "客户已保存";
    await refresh();
  } catch (error) {
    formMessage.textContent = error.message;
  }
});

document.querySelector("#preview-import").addEventListener("click", async () => {
  try {
    previewRecords = JSON.parse(importInput.value);
    const result = await request("/api/customers/import/preview", { method: "POST", body: JSON.stringify(previewRecords) });
    const duplicates = result.duplicate_phones || [];
    const existing = result.existing_phones || [];
    previewResult.textContent = duplicates.length || existing.length ? `重复手机号：${[...new Set([...duplicates, ...existing])].join("、")}` : "未发现重复手机号，可以提交";
    commitButton.disabled = Boolean(duplicates.length || existing.length);
  } catch (error) {
    previewRecords = null;
    commitButton.disabled = true;
    previewResult.textContent = error.message;
  }
});

commitButton.addEventListener("click", async () => {
  if (!previewRecords) return;
  try {
    await request("/api/customers/import", { method: "POST", body: JSON.stringify(previewRecords) });
    previewResult.textContent = "导入完成";
    commitButton.disabled = true;
    await refresh();
  } catch (error) {
    previewResult.textContent = error.message;
  }
});

document.querySelector("#refresh").addEventListener("click", refresh);
document.querySelector("#refresh-logs").addEventListener("click", refresh);
refresh().catch((error) => { formMessage.textContent = error.message; });
