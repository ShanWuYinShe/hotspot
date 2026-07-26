/**
 * CreateGroupSheet — a bottom sheet / modal for creating new groups.
 * Used from: NearbyGroups (FAB), Profile > 我的群组 (header button),
 *            and as "创建群聊" entry from NewConversationSheet.
 */
import React, { useState } from "react";
import { X, Camera, Globe, Lock, Check, ChevronRight, MapPin } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useHoverConfig } from "../../hooks/useHoverConfig";

const CATEGORIES = ["摄影", "音乐", "户外", "阅读", "美食", "健身", "游戏", "其他"];

interface Props {
  onClose: () => void;
  onCreated?: (name: string) => void;
}

export function CreateGroupSheet({ onClose, onCreated }: Props) {
  const hover = useHoverConfig();
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");
  const [category, setCategory] = useState("摄影");
  const [isPrivate, setIsPrivate] = useState(false);
  const [step, setStep] = useState<"form" | "done">("form");

  const canCreate = name.trim().length >= 2;

  const handleCreate = () => {
    if (!canCreate) return;
    setStep("done");
    onCreated?.(name.trim());
  };

  return (
    <div className="flex flex-col h-full" style={{ background: "var(--neu-bg)" }}>
      {/* Handle */}
      <div className="flex justify-center pt-3 pb-1 flex-shrink-0">
        <div className="rounded-full" style={{ width: 36, height: 4, background: "var(--neu-border)" }} />
      </div>

      {/* Header */}
      <div className="flex items-center justify-between px-5 py-3 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <h2 style={{ fontSize: 17, color: "var(--neu-text)" }}>创建群组</h2>
        <motion.button whileTap={hover.tap as any} onClick={onClose}
          className="flex items-center justify-center rounded-full outline-none"
          style={{ width: 32, height: 32, background: "var(--neu-bg)", boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <X style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={2} />
        </motion.button>
      </div>

      <AnimatePresence mode="wait">
        {step === "done" ? (
          <motion.div key="done"
            initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }}
            className="flex-1 flex flex-col items-center justify-center gap-4 px-8 text-center">
            <motion.div
              initial={{ scale: 0 }} animate={{ scale: 1 }}
              transition={{ type: "spring", stiffness: 320, damping: 18, delay: 0.1 }}
              className="flex items-center justify-center rounded-full"
              style={{ width: 72, height: 72, background: "var(--neu-accent-primary)", boxShadow: "0 8px 24px var(--neu-glow)" }}>
              <Check style={{ width: 32, height: 32, color: "white" }} strokeWidth={2.5} />
            </motion.div>
            <div>
              <p style={{ fontSize: 20, color: "var(--neu-text)", fontWeight: 600 }}>群组已创建！</p>
              <p style={{ fontSize: 13, color: "var(--neu-text-muted)", marginTop: 6 }}>「{name}」已成功创建，现在可以邀请好友加入了</p>
            </div>
            <motion.button whileTap={hover.tap as any} onClick={onClose}
              className="px-8 py-3 rounded-full outline-none"
              style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 15, fontWeight: 500, boxShadow: "0 6px 20px var(--neu-glow)" }}>
              完成
            </motion.button>
          </motion.div>
        ) : (
          <motion.div key="form"
            initial={{ opacity: 0 }} animate={{ opacity: 1 }}
            className="flex-1 overflow-y-auto no-scrollbar px-5 pt-4 pb-8 flex flex-col gap-5">

            {/* Cover photo selector */}
            <div className="flex justify-center">
              <motion.button whileTap={hover.tap as any}
                className="flex flex-col items-center justify-center gap-2 rounded-[22px] outline-none"
                style={{ width: "100%", height: 110, background: "var(--neu-bg)", boxShadow: "inset 4px 4px 10px var(--neu-shadow-dark), inset -4px -4px 10px var(--neu-shadow-light)", border: "2px dashed var(--neu-border)" }}>
                <Camera style={{ width: 26, height: 26, color: "var(--neu-text-muted)" }} strokeWidth={1.25} />
                <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>添加封面图片</span>
              </motion.button>
            </div>

            {/* Group name */}
            <div>
              <label style={{ fontSize: 11, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--neu-text-muted)" }}>群组名称 *</label>
              <div className="flex items-center gap-2.5 mt-1.5 px-4 rounded-[14px] neu-input-surface" style={{ height: 46 }}>
                <input className="flex-1 bg-transparent outline-none"
                  style={{ fontSize: 14, color: "var(--neu-text)" }}
                  placeholder="给你的群组起个名字…"
                  value={name}
                  maxLength={30}
                  onChange={e => setName(e.target.value)}
                />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)", flexShrink: 0 }}>{name.length}/30</span>
              </div>
            </div>

            {/* Description */}
            <div>
              <label style={{ fontSize: 11, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--neu-text-muted)" }}>简介</label>
              <div className="mt-1.5 px-4 py-3 rounded-[14px] neu-input-surface">
                <textarea className="w-full bg-transparent outline-none resize-none"
                  style={{ fontSize: 13, color: "var(--neu-text)", minHeight: 72 }}
                  placeholder="描述一下这个群组的主题…"
                  value={desc}
                  maxLength={150}
                  onChange={e => setDesc(e.target.value)}
                />
                <div className="flex justify-end">
                  <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{desc.length}/150</span>
                </div>
              </div>
            </div>

            {/* Category chips */}
            <div>
              <label style={{ fontSize: 11, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--neu-text-muted)" }}>分类</label>
              <div className="flex flex-wrap gap-2 mt-2">
                {CATEGORIES.map(cat => (
                  <motion.button key={cat} whileTap={hover.tap as any}
                    onClick={() => setCategory(cat)}
                    className="px-3.5 py-1.5 rounded-full outline-none"
                    style={category === cat
                      ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, fontWeight: 500, boxShadow: "0 3px 8px var(--neu-glow)" }
                      : { boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", color: "var(--neu-text-muted)", background: "var(--neu-bg)", fontSize: 12 }
                    }
                  >
                    {cat}
                  </motion.button>
                ))}
              </div>
            </div>

            {/* Privacy toggle */}
            <div className="flex items-center justify-between px-4 py-3.5 rounded-[16px]"
              style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
              <div className="flex items-center gap-3">
                {isPrivate
                  ? <Lock style={{ width: 18, height: 18, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                  : <Globe style={{ width: 18, height: 18, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                }
                <div>
                  <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{isPrivate ? "私密群组" : "公开群组"}</p>
                  <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                    {isPrivate ? "仅受邀请的人可加入" : "所有人都可以找到并加入"}
                  </p>
                </div>
              </div>
              <motion.button whileTap={{ scale: 0.92 }} onClick={() => setIsPrivate(v => !v)}
                className="flex-shrink-0 rounded-full outline-none"
                style={{
                  width: 48, height: 28,
                  background: isPrivate ? "var(--neu-accent-primary)" : "rgba(120,120,128,0.25)",
                  position: "relative",
                  transition: "background 0.22s ease",
                  boxShadow: isPrivate ? "inset 0 1px 2px rgba(0,0,0,0.12)" : "inset 0 1px 2px rgba(0,0,0,0.08)",
                }}>
                <motion.div animate={{ x: isPrivate ? 22 : 2 }}
                  transition={{ type: "spring", stiffness: 420, damping: 30 }}
                  style={{ position: "absolute", top: 3, width: 22, height: 22, borderRadius: 11, background: "#fff", boxShadow: "0 1px 4px rgba(0,0,0,0.26)" }} />
              </motion.button>
            </div>

            {/* Location */}
            <div className="flex items-center gap-2.5 px-4 py-3 rounded-[16px]"
              style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
              <MapPin style={{ width: 16, height: 16, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={1.5} />
              <div className="flex-1 min-w-0">
                <p style={{ fontSize: 13, color: "var(--neu-text)" }}>徐州市 · 云龙区</p>
                <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>自动定位 — 可修改</p>
              </div>
              <ChevronRight style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </div>

            {/* Create button */}
            <motion.button whileTap={hover.tap as any}
              onClick={handleCreate}
              disabled={!canCreate}
              className="flex items-center justify-center gap-2 w-full py-4 rounded-[18px] outline-none"
              style={{
                background: canCreate ? "var(--neu-accent-primary)" : "var(--neu-bg)",
                color: canCreate ? "white" : "var(--neu-text-muted)",
                fontSize: 15, fontWeight: 500,
                boxShadow: canCreate
                  ? "0 6px 22px var(--neu-glow)"
                  : "inset 4px 4px 10px var(--neu-shadow-dark), inset -4px -4px 10px var(--neu-shadow-light)",
                border: canCreate ? "none" : "1px solid var(--neu-border)",
                transition: "all 0.22s ease",
                cursor: canCreate ? "pointer" : "not-allowed",
              }}>
              创建群组
            </motion.button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
