import { useState, useCallback } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Plus, Trash2, Megaphone } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { toast } from "sonner";
import { adminAnnouncementsService } from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { usePagination } from "@/lib/hooks/use-pagination";
import type {
  AdminAnnouncementItem,
  AnnouncementSeverity,
} from "@/lib/types/api";

const SEVERITY_OPTIONS: {
  label: string;
  value: AnnouncementSeverity;
  color: string;
}[] = [
  { label: "通知", value: "info", color: "#6366F1" },
  { label: "警告", value: "warning", color: "#F59E0B" },
  { label: "严重", value: "error", color: "#EF4444" },
];

function severityColor(s: AnnouncementSeverity): string {
  return SEVERITY_OPTIONS.find((o) => o.value === s)?.color ?? "#6366F1";
}

function severityLabel(s: AnnouncementSeverity): string {
  return SEVERITY_OPTIONS.find((o) => o.value === s)?.label ?? s;
}

export function AdminAnnouncements() {
  const navigate = useNavigate();
  useDocumentTitle("公告管理");

  const page = usePagination<AdminAnnouncementItem>({
    uniqueKey: "announcement_id",
  });
  const [showForm, setShowForm] = useState(false);
  const [formTitle, setFormTitle] = useState("");
  const [formContent, setFormContent] = useState("");
  const [formSeverity, setFormSeverity] =
    useState<AnnouncementSeverity>("info");
  const [submitting, setSubmitting] = useState(false);

  const fetchList = useCallback(
    (refresh = true) => {
      page.load(
        (cursor) =>
          adminAnnouncementsService.search({
            pagination: { limit: 20, cursor },
          }),
        refresh,
      );
    },
    [page],
  );

  const handleCreate = useCallback(async () => {
    if (!formTitle.trim() || !formContent.trim()) {
      toast.error("请填写标题和内容");
      return;
    }
    setSubmitting(true);
    try {
      const res = await adminAnnouncementsService.create({
        title: formTitle.trim(),
        content: formContent.trim(),
        severity: formSeverity,
      });
      if (isSuccess(res)) {
        toast.success("公告已创建");
        setFormTitle("");
        setFormContent("");
        setFormSeverity("info");
        setShowForm(false);
        fetchList(true);
      } else {
        toast.error("创建失败");
      }
    } catch {
      toast.error("创建请求失败");
    } finally {
      setSubmitting(false);
    }
  }, [formTitle, formContent, formSeverity, fetchList]);

  const handleDelete = useCallback(
    async (item: AdminAnnouncementItem) => {
      const confirmed = window.confirm(`确定要删除公告「${item.title}」吗？`);
      if (!confirmed) return;

      try {
        const res = await adminAnnouncementsService.delete({
          announcement_id: item.announcement_id,
        });
        if (isSuccess(res)) {
          toast.success("已删除");
          fetchList(true);
        } else {
          toast.error("删除失败");
        }
      } catch {
        toast.error("删除请求失败");
      }
    },
    [fetchList],
  );

  return (
    <div
      className="flex flex-col min-h-screen"
      style={{ background: "var(--neu-bg)" }}
    >
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: "var(--neu-text)" }}>
          公告管理
        </p>
        <div className="flex-1" />
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => setShowForm(!showForm)}
          className="flex items-center justify-center rounded-full"
          style={{
            width: 32,
            height: 32,
            background: showForm ? "var(--neu-danger)" : "var(--neu-accent-primary)",
          }}
        >
          <Plus
            style={{
              width: 16,
              height: 16,
              color: "white",
              transform: showForm ? "rotate(45deg)" : "none",
              transition: "transform 0.2s",
            }}
            strokeWidth={2}
          />
        </motion.button>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex-1 px-4 py-4"
        style={{
          maxWidth: 600,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {/* Create form */}
        <AnimatePresence>
          {showForm && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.25 }}
              className="overflow-hidden mb-4"
            >
              <div
                className="px-4 py-4 rounded-[20px]"
                style={{
                  background: "var(--neu-bg)",
                  boxShadow:
                    "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                <p
                  className="mb-3"
                  style={{
                    fontSize: 14,
                    fontWeight: 600,
                    color: "var(--neu-text)",
                  }}
                >
                  新建公告
                </p>

                <input
                  value={formTitle}
                  onChange={(e) => setFormTitle(e.target.value)}
                  placeholder="公告标题"
                  className="w-full bg-transparent outline-none mb-3 px-3 py-2 rounded-[12px]"
                  style={{
                    fontSize: 14,
                    color: "var(--neu-text)",
                    border: "1px solid var(--neu-border)",
                  }}
                />

                <textarea
                  value={formContent}
                  onChange={(e) => setFormContent(e.target.value)}
                  placeholder="公告内容"
                  rows={3}
                  className="w-full bg-transparent outline-none mb-3 px-3 py-2 rounded-[12px] resize-none"
                  style={{
                    fontSize: 14,
                    color: "var(--neu-text)",
                    border: "1px solid var(--neu-border)",
                  }}
                />

                {/* Severity picker */}
                <div className="flex gap-2 mb-3">
                  {SEVERITY_OPTIONS.map((opt) => (
                    <motion.button
                      key={opt.value}
                      whileTap={{ scale: 0.94 }}
                      onClick={() => setFormSeverity(opt.value)}
                      className="px-3 py-1 rounded-full"
                      style={{
                        fontSize: 12,
                        background:
                          formSeverity === opt.value
                            ? `${opt.color}20`
                            : "transparent",
                        color:
                          formSeverity === opt.value
                            ? opt.color
                            : "var(--neu-text-muted)",
                        border: `1px solid ${formSeverity === opt.value ? opt.color : "var(--neu-border)"}`,
                      }}
                    >
                      {opt.label}
                    </motion.button>
                  ))}
                </div>

                <motion.button
                  whileTap={{ scale: 0.96 }}
                  onClick={handleCreate}
                  disabled={submitting}
                  className="w-full py-2.5 rounded-[12px]"
                  style={{
                    background: "var(--neu-accent-primary)",
                    color: "var(--text-on-accent, white)",
                    fontSize: 14,
                    opacity: submitting ? 0.6 : 1,
                  }}
                >
                  {submitting ? "创建中..." : "发布公告"}
                </motion.button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {page.error && (
          <p
            className="mb-3 text-center"
            style={{ fontSize: 13, color: "var(--neu-danger)" }}
          >
            {page.error}
          </p>
        )}

        {page.loading && (
          <div className="flex items-center justify-center py-16">
            <div
              className="rounded-full"
              style={{
                width: 28,
                height: 28,
                border: "3px solid var(--neu-border)",
                borderTopColor: "var(--neu-accent-primary)",
                animation: "spin 0.8s linear infinite",
              }}
            />
            <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
          </div>
        )}

        {!page.loading && page.items.length === 0 && (
          <div className="flex flex-col items-center py-16">
            <Megaphone
              style={{ width: 40, height: 40, color: "var(--neu-text-muted)" }}
              strokeWidth={1.2}
            />
            <p
              className="mt-3"
              style={{ fontSize: 14, color: "var(--neu-text-muted)" }}
            >
              暂无公告
            </p>
          </div>
        )}

        <div className="flex flex-col gap-2">
          {page.items.map((ann) => (
            <div
              key={ann.announcement_id}
              className="px-4 py-3 rounded-[16px]"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <div className="flex items-center justify-between mb-1.5">
                <div className="flex items-center gap-2">
                  <span
                    className="px-2 py-0.5 rounded-full"
                    style={{
                      fontSize: 11,
                      background: `${severityColor(ann.severity)}20`,
                      color: severityColor(ann.severity),
                    }}
                  >
                    {severityLabel(ann.severity)}
                  </span>
                  <span
                    className="truncate"
                    style={{
                      fontSize: 14,
                      fontWeight: 500,
                      color: "var(--neu-text)",
                      maxWidth: 240,
                    }}
                  >
                    {ann.title}
                  </span>
                </div>
                <motion.button
                  whileTap={{ scale: 0.9 }}
                  onClick={() => handleDelete(ann)}
                  className="flex items-center justify-center rounded-full flex-shrink-0"
                  style={{ width: 28, height: 28, background: "var(--neu-danger-soft)" }}
                >
                  <Trash2
                    style={{ width: 13, height: 13, color: "var(--neu-danger)" }}
                    strokeWidth={1.5}
                  />
                </motion.button>
              </div>
              <p
                className="line-clamp-2 mb-1.5"
                style={{ fontSize: 12, color: "var(--neu-text-muted)" }}
              >
                {ann.content}
              </p>
              <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                {new Date(ann.created_at).toLocaleString()}
                {" · 创建者 "}
                {ann.created_by?.nickname ?? "系统"}
              </p>
            </div>
          ))}
        </div>

        {page.hasMore && !page.loading && page.items.length > 0 && (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => fetchList(false)}
            disabled={page.loadingMore}
            className="mt-4 py-2.5 rounded-[14px] w-full"
            style={{
              background: "var(--neu-bg)",
              boxShadow:
                "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
              fontSize: 13,
              color: "var(--neu-text-muted)",
            }}
          >
            {page.loadingMore ? "加载中..." : "加载更多"}
          </motion.button>
        )}
      </motion.div>
    </div>
  );
}
