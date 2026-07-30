import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Camera, User, FileText, Check } from "lucide-react";
import { ActionIconButton } from "../../components/shared/ActionIconButton";
import { motion, AnimatePresence } from "motion/react";
import { getAccount, updateAccount } from "../../../api";

const TAG_OPTIONS = ["摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术", "科技", "游戏", "户外", "时尚", "电影"];

function InputRow({
  label, icon: Icon, value, onChange, placeholder, multiline = false, maxLen,
}: {
  label: string; icon: React.ElementType; value: string;
  onChange: (v: string) => void; placeholder: string;
  multiline?: boolean; maxLen?: number;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-1.5">
        <Icon style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
        <label className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: "0.1em" }}>
          {label}
        </label>
      </div>
      <div className="px-4 py-3 rounded-[18px] neu-input-surface">
        {multiline ? (
          <div>
            <textarea
              value={value}
              onChange={e => onChange(e.target.value.slice(0, maxLen))}
              placeholder={placeholder}
              rows={3}
              className="w-full bg-transparent outline-none resize-none text-[var(--neu-text)]"
              style={{ fontSize: 14, lineHeight: 1.65 }}
            />
            {maxLen && (
              <div className="flex justify-end mt-1">
                <span className={value.length > maxLen * 0.85 ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"} style={{ fontSize: 11 }}>
                  {value.length}/{maxLen}
                </span>
              </div>
            )}
          </div>
        ) : (
          <input value={value} onChange={e => onChange(e.target.value.slice(0, maxLen))}
            placeholder={placeholder} className="w-full bg-transparent outline-none text-[var(--neu-text)]"
            style={{ fontSize: 14 }} />
        )}
      </div>
    </div>
  );
}

export function EditProfilePage() {
  const navigate = useNavigate();
  const [nickname, setNickname] = useState("");
  const [bio, setBio] = useState("");
  const [tags, setTags] = useState<Set<string>>(new Set());
  const [saved, setSaved] = useState(false);
  const [avatar, setAvatar] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function fetchAccount() {
      try {
        const res = await getAccount().catch(() => ({ code: -1, data: null }));
        if (cancelled) return;
        const data = (res.code === 0 ? res.data : null) as any;
        if (data) {
          // spec: AccountData.nickname, .bio, .tags, .avatar_url
          setNickname(data.nickname ?? data.name ?? "");    // "name" is legacy, not in spec
          setBio(data.bio ?? "");
          setAvatar(data.avatar_url ?? data.avatar ?? "");
          if (data.tags) setTags(new Set(data.tags));
        }
      } finally {
        // ignore fetch error
      }
    }
    fetchAccount();
    return () => { cancelled = true; };
  }, []);

  const toggleTag = (t: string) => setTags(prev => { const n = new Set(prev); n.has(t) ? n.delete(t) : n.add(t); return n; });

  const handleSave = async () => {
    try {
      // spec: updateAccount { nickname?, avatar_file_id?, bio?, tags?, gender? }
      await updateAccount({
        nickname: nickname || undefined,
        bio: bio || undefined,
        tags: tags.size > 0 ? Array.from(tags) : undefined,
      });
    } catch {
      // ignore save error
    }
    setSaved(true);
    setTimeout(() => { setSaved(false); navigate(-1); }, 800);
  };

  return (
    <>
      {/* Header */}
      <div className="sticky top-0 z-10 flex items-center justify-between px-5 py-3 bg-[var(--neu-bg)]"
        style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-3">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} />
          <h1 className="text-[var(--neu-text)]" style={{ fontSize: 18 }}>编辑资料</h1>
        </div>
        <motion.button whileTap={{ scale: 0.93 }} onClick={handleSave}
          className={`flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all ${saved ? "bg-[var(--semantic-online)]" : "bg-[var(--neu-accent-primary)]"} text-white`}
          style={{ fontSize: 13, boxShadow: `0 4px 14px ${saved ? "color-mix(in srgb, var(--semantic-online) 35%, transparent)" : "var(--neu-glow)"}`, border: "none" }}>
          <AnimatePresence mode="wait" initial={false}>
            {saved
              ? <motion.span key="check" initial={{ scale: 0 }} animate={{ scale: 1 }} exit={{ scale: 0 }} style={{ display: "flex" }}>
                  <Check style={{ width: 14, height: 14, color: "white" }} strokeWidth={2.5} />
                </motion.span>
              : <motion.span key="save" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>保存</motion.span>
            }
          </AnimatePresence>
        </motion.button>
      </div>

      <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28 }}
        className="content-mode-reading flex flex-col px-5 pt-4 pb-10 gap-5"
        style={{ maxWidth: "var(--content-reading-max)", marginLeft: "auto", marginRight: "auto", width: "100%" }}>

        {/* Avatar section */}
        <div className="relative rounded-[24px] overflow-hidden" style={{ height: "clamp(120px, 18vw, 160px)" }}>
          <div className="absolute inset-0 bg-[var(--neu-surface-flat)]" />
          <motion.button whileTap={{ scale: 0.9 }}
            className="absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none bg-[var(--overlay-strong)] text-white"
            style={{ padding: "6px 12px", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)", fontSize: 12 }}>
            <Camera style={{ width: 13, height: 13 }} strokeWidth={1.75} />
            更换横幅
          </motion.button>
        </div>

        {/* Avatar edit */}
        <div className="flex items-center gap-4 -mt-8 ml-2">
          <div className="relative flex-shrink-0">
            <div className="rounded-[22px] overflow-hidden"
              style={{ width: 72, height: 72, border: "3px solid var(--neu-bg)", boxShadow: "0 6px 20px rgba(0,0,0,0.22)" }}>
              <img src={avatar} alt="avatar" className="w-full h-full object-cover" />
            </div>
            <motion.button whileTap={{ scale: 0.9 }}
              className="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)]"
              style={{ width: 26, height: 26, border: "2px solid var(--neu-bg)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
              <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
            </motion.button>
          </div>
          <div>
            <p className="text-[var(--neu-text)]" style={{ fontSize: 16 }}>{nickname || "未设置昵称"}</p>
          </div>
        </div>

        {/* Form fields — spec: nickname, bio, tags are the only user-editable profile fields. */}
        <InputRow label="昵称" icon={User} value={nickname} onChange={setNickname} placeholder="你的名字" maxLen={30} />
        <InputRow label="个人简介" icon={FileText} value={bio} onChange={setBio} placeholder="介绍一下自己…" multiline maxLen={150} />

        {/* Interest tags */}
        <div className="flex flex-col gap-2.5">
          <div className="flex items-center gap-1.5">
            <label className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: "0.1em" }}>兴趣标签</label>
            <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, opacity: 0.6 }}>({tags.size}/5)</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {TAG_OPTIONS.map(t => {
              const active = tags.has(t);
              const disabled = !active && tags.size >= 5;
              return (
                <motion.button key={t} whileTap={{ scale: 0.92 }}
                  onClick={() => !disabled && toggleTag(t)}
                  className={`px-3.5 py-1.5 rounded-full outline-none transition-all ${active ? "bg-[var(--neu-accent-primary)] text-white" : "bg-[var(--neu-bg)] text-[var(--neu-text-muted)]"}`}
                  style={active
                    ? { fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)", border: "none" }
                    : { fontSize: 12, border: "1px solid var(--neu-border)", opacity: disabled ? 0.4 : 1, boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
                  #{t}
                </motion.button>
              );
            })}
          </div>
        </div>

        {/* Danger zone */}
        <div className="mt-3">
          <p className="uppercase mb-3 text-[var(--neu-text-muted)]" style={{ fontSize: 10, letterSpacing: "0.14em" }}>账号</p>
          <motion.button whileTap={{ scale: 0.98 }} className="w-full text-left px-4 py-3.5 rounded-[18px] outline-none bg-[var(--neu-bg)] text-[var(--semantic-hot)]"
            style={{ boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", fontSize: 14 }}>
            注销账号
          </motion.button>
        </div>
      </motion.div>
    </>
  );
}
