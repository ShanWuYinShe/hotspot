import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Camera, MapPin, Link, User, FileText, Check } from "lucide-react";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { motion, AnimatePresence } from "motion/react";
import { getAccount, updateAccount } from "../../api";

const AVATAR = "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400";
const BANNER = "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=1080";

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
        <label style={{ fontSize: 11, color: "var(--neu-text-muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
          {label}
        </label>
      </div>
      {/* neu-input-surface: convex at rest, concave on focus-within */}
      <div className="px-4 py-3 rounded-[18px] neu-input-surface">
        {multiline ? (
          <div>
            <textarea
              value={value}
              onChange={e => onChange(e.target.value.slice(0, maxLen))}
              placeholder={placeholder}
              rows={3}
              className="w-full bg-transparent outline-none resize-none"
              style={{ fontSize: 14, color: "var(--neu-text)", lineHeight: 1.65 }}
            />
            {maxLen && (
              <div className="flex justify-end mt-1">
                <span style={{ fontSize: 11, color: value.length > maxLen * 0.85 ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                  {value.length}/{maxLen}
                </span>
              </div>
            )}
          </div>
        ) : (
          <input value={value} onChange={e => onChange(e.target.value.slice(0, maxLen))}
            placeholder={placeholder} className="w-full bg-transparent outline-none"
            style={{ fontSize: 14, color: "var(--neu-text)" }} />
        )}
      </div>
    </div>
  );
}

export function EditProfile() {
  const navigate = useNavigate();
  const [name,     setName]     = useState("");
  const [handle,   setHandle]   = useState("");
  const [bio,      setBio]      = useState("");
  const [location, setLocation] = useState("");
  const [website,  setWebsite]  = useState("");
  const [tags,     setTags]     = useState<Set<string>>(new Set());
  const [saved,    setSaved]    = useState(false);
  const [avatar,   setAvatar]   = useState(AVATAR);
  const [banner,   setBanner]   = useState(BANNER);
  const [loading,  setLoading]  = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function fetchAccount() {
      try {
        setLoading(true);
        const res = await getAccount().catch(() => ({ code: -1, data: null }));
        if (cancelled) return;
        const data = (res.code === 0 ? res.data : null) as any;
        if (data) {
          setName(data.name ?? data.nickname ?? "");
          setHandle((data.handle ?? data.username ?? "").replace("@", ""));
          setBio(data.bio ?? data.description ?? "");
          setLocation(data.location ?? data.place ?? "");
          setWebsite(data.website ?? data.url ?? "");
          setAvatar(data.avatar ?? AVATAR);
          setBanner(data.banner ?? data.cover ?? BANNER);
          if (data.tags) {
            setTags(new Set(data.tags));
          }
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchAccount();
    return () => { cancelled = true; };
  }, []);

  const toggleTag = (t: string) => setTags(prev => { const n = new Set(prev); n.has(t) ? n.delete(t) : n.add(t); return n; });

  const handleSave = async () => {
    try {
      await updateAccount({
        name,
        handle,
        bio,
        location,
        website,
        tags: Array.from(tags),
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
      <div className="sticky top-0 z-10 flex items-center justify-between px-5 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-3">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} />
          <h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>编辑资料</h1>
        </div>
        <motion.button whileTap={{ scale: 0.93 }} onClick={handleSave}
          className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all"
          style={{ background: saved ? "var(--semantic-online)" : "var(--neu-accent-primary)", color: "white", fontSize: 13, boxShadow: `0 4px 14px ${saved ? "color-mix(in srgb, var(--semantic-online) 35%, transparent)" : "var(--neu-glow)"}`, border: "none" }}>
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
          <img src={banner} alt="banner" className="absolute inset-0 w-full h-full object-cover" />
          <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, transparent 40%, rgba(0,0,0,0.5))" }} />
          <motion.button whileTap={{ scale: 0.9 }}
            className="absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none"
            style={{ padding: "6px 12px", background: "var(--overlay-strong)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)", fontSize: 12, color: "white" }}>
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
              className="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
              style={{ width: 26, height: 26, background: "var(--neu-accent-primary)", border: "2px solid var(--neu-bg)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
              <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
            </motion.button>
          </div>
          <div>
            <p style={{ fontSize: 16, color: "var(--neu-text)" }}>{name}</p>
            <p style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>@{handle}</p>
          </div>
        </div>

        {/* Form fields */}
        <InputRow label="姓名" icon={User} value={name} onChange={setName} placeholder="你的名字" maxLen={30} />
        <InputRow label="用户名" icon={User} value={handle} onChange={v => setHandle(v.replace(/\s/g, "").toLowerCase())} placeholder="用户名 (不含@)" maxLen={20} />
        <InputRow label="个人简介" icon={FileText} value={bio} onChange={setBio} placeholder="介绍一下自己…" multiline maxLen={150} />
        <InputRow label="所在地" icon={MapPin} value={location} onChange={setLocation} placeholder="城市或地区" maxLen={40} />
        <InputRow label="网站" icon={Link} value={website} onChange={setWebsite} placeholder="https://…" maxLen={100} />

        {/* Interest tags */}
        <div className="flex flex-col gap-2.5">
          <div className="flex items-center gap-1.5">
            <label style={{ fontSize: 11, color: "var(--neu-text-muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>兴趣标签</label>
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)", opacity: 0.6 }}>({tags.size}/5)</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {TAG_OPTIONS.map(t => {
              const active = tags.has(t);
              const disabled = !active && tags.size >= 5;
              return (
                <motion.button key={t} whileTap={{ scale: 0.92 }}
                  onClick={() => !disabled && toggleTag(t)}
                  className="px-3.5 py-1.5 rounded-full outline-none transition-all"
                  style={active
                    ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)", border: "none" }
                    : { background: "var(--neu-bg)", color: disabled ? "var(--neu-text-muted)" : "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)", opacity: disabled ? 0.4 : 1, boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
                  #{t}
                </motion.button>
              );
            })}
          </div>
        </div>

        {/* Danger zone */}
        <div className="mt-3">
          <p className="uppercase mb-3" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>账号</p>
          <motion.button whileTap={{ scale: 0.98 }} className="w-full text-left px-4 py-3.5 rounded-[18px] outline-none"
            style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", fontSize: 14, color: "var(--semantic-hot)" }}>
            注销账号
          </motion.button>
        </div>
      </motion.div>
    </>
  );
}