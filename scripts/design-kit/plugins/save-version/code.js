// 最小插件：只写 Figma 版本历史，不改任何画布节点。
// 运行：Figma 桌面端 → 右键画布 → Plugins → Development → "HotSpot Save Design Version"
// 每次要打新记录时改下方 VERSION_TITLE / VERSION_DESCRIPTION 两处即可。
var VERSION_TITLE = '细节精修版 · 胶囊对比度与主题绑定修复 (11月18日 19:00)';
var VERSION_DESCRIPTION = [
  '【本次版本更新内容摘要】',
  '',
  '一、胶囊底透明度治理（三端 41 处）：图片计数/已关注/加入等胶囊底为「深色变量 + 38%/55% 透明度」中间态，在浅色玻璃条上渲染成浅灰底白字不可读；统一改为 surface-overlay 不透明填充（opacity=1，全模式深底），白字保持 content-inverse 反白。',
  '',
  '二、胶囊内图标-文字间距重排（15 处）：按复合胶囊居中公式（图标 + 5px + 文字，整体光学居中）。',
  '',
  '三、语义绑定纠错（15 处）：叠字卡片黑胶囊内作者名误绑 text-primary 改绑 content-inverse；信息流/附近枢纽 13 个位置标签（云龙湖东岸等）误绑 content-inverse 改绑 text-secondary。',
  '',
  '四、规范约束固化：巡检脚本新增 pill-bg-translucent 规则（胶囊底 20%~70% 中间态透明度 + 浅色文字即违规），on-soft 状态档（≤20% 透明 + 深字）不受影响。',
  '',
  '五、质量门禁：全库巡检 violations 0 / warnings 0；搜索页/首页放大裁剪逐卡核验；vibrant-dark 模式切换截图验证无浅底白字回归后复原。',
].join('\n');

async function main() {
  if (typeof figma.saveVersionHistoryAsync !== 'function') {
    figma.closePlugin('当前环境不支持 saveVersionHistoryAsync');
    return;
  }
  try {
    await figma.saveVersionHistoryAsync(VERSION_TITLE, VERSION_DESCRIPTION);
    figma.notify('已写入版本历史：' + VERSION_TITLE, { timeout: 5000 });
    figma.closePlugin();
  } catch (e) {
    figma.closePlugin('保存版本失败：' + (e && e.message ? e.message : e));
  }
}
main();
