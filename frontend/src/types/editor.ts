/**
 * 全局类型定义。
 * 后续在此添加：
 *   - 编辑器 Block 的类型
 *   - 自定义 Node attributes 的类型
 *   - Strapi CMS 的 media 响应类型
 *   - Markdown AST 相关类型
 */

export interface EditorBlock {
  id: string;
  type: string;
  attrs?: Record<string, unknown>;
  content?: EditorBlock[];
}
