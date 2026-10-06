// @/api 统一出口：直接透传 client facade 的全部导出（client.ts 已按域拆分，
// facade 保证导出面；显式白名单此前已发生漂移——listReports 等被迫绕行 import）。
// 新增端点只需放入对应 domains/*.ts，无需改本文件。
export { Endpoints } from '@/api/endpoints';
export * from '@/api/client';
