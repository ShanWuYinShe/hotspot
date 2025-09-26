/// 智能内容安全审核服务
/// 
/// 功能职责：
/// - 实现多层次内容安全检测
/// - 支持敏感词过滤、语义分析、图像识别
/// - 提供风险评分和处理建议
/// - 支持机器学习模型集成

use std::collections::HashMap;
use tracing::{debug, warn};
use serde::{Serialize, Deserialize};
use regex::Regex;

/// 内容安全风险等级
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum SecurityRiskLevel {
    /// 安全内容
    Safe,
    /// 低风险 - 可能需要人工审核
    Low,
    /// 中等风险 - 建议限制传播
    Medium,
    /// 高风险 - 建议屏蔽
    High,
    /// 极高风险 - 立即屏蔽
    Critical,
}

/// 内容类型
#[derive(Debug, Clone, PartialEq)]
pub enum ContentType {
    /// 文本内容
    Text,
    /// 图片内容
    Image,
    /// 用户资料
    UserProfile,
    /// 群组信息
    GroupInfo,
}

/// 安全检测结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityCheckResult {
    /// 风险等级
    pub risk_level: SecurityRiskLevel,
    /// 风险分数 (0-100)
    pub risk_score: f64,
    /// 检测到的问题类型
    pub detected_issues: Vec<String>,
    /// 处理建议
    pub action_recommendation: String,
    /// 是否需要人工审核
    pub requires_manual_review: bool,
    /// 检测详情
    pub details: HashMap<String, serde_json::Value>,
}

/// 敏感词配置
#[derive(Debug, Clone)]
struct SensitiveWordConfig {
    /// 严重敏感词 (直接屏蔽)
    critical_words: Vec<String>,
    /// 一般敏感词 (需要审核)
    moderate_words: Vec<String>,
    /// 轻微敏感词 (降低权重)
    mild_words: Vec<String>,
    /// 正则表达式模式
    patterns: Vec<Regex>,
}

/// 内容安全审核服务
pub struct ContentSecurityService {
    /// 敏感词配置
    sensitive_words: SensitiveWordConfig,
    /// 是否启用严格模式
    strict_mode: bool,
}

impl ContentSecurityService {
    /// 创建新的内容安全服务
    pub fn new() -> Self {
        let sensitive_words = Self::load_sensitive_words();
        
        Self {
            sensitive_words,
            strict_mode: true, // 默认启用严格模式
        }
    }
    
    /// 加载敏感词配置
    fn load_sensitive_words() -> SensitiveWordConfig {
        // 严重敏感词 - 涉及政治、暴力、色情等
        let critical_words = vec![
            // 政治敏感
            "习近平".to_string(),
            "共产党".to_string(),
            "六四".to_string(),
            "天安门".to_string(),
            "法轮功".to_string(),
            "台独".to_string(),
            "藏独".to_string(),
            "疆独".to_string(),
            
            // 暴力恐怖
            "杀人".to_string(),
            "爆炸".to_string(),
            "恐怖主义".to_string(),
            "自杀".to_string(),
            "血腥".to_string(),
            
            // 色情低俗
            "色情".to_string(),
            "裸体".to_string(),
            "性交".to_string(),
            "卖淫".to_string(),
            
            // 违法犯罪
            "毒品".to_string(),
            "贩毒".to_string(),
            "洗钱".to_string(),
            "诈骗".to_string(),
            "赌博".to_string(),
        ];
        
        // 一般敏感词 - 需要人工审核
        let moderate_words = vec![
            "死".to_string(),
            "杀".to_string(),
            "血".to_string(),
            "暴力".to_string(),
            "仇恨".to_string(),
            "歧视".to_string(),
            "种族".to_string(),
            "宗教".to_string(),
            "政治".to_string(),
            "抗议".to_string(),
            "示威".to_string(),
            "革命".to_string(),
        ];
        
        // 轻微敏感词 - 降低内容权重
        let mild_words = vec![
            "笨蛋".to_string(),
            "白痴".to_string(),
            "傻子".to_string(),
            "垃圾".to_string(),
            "废物".to_string(),
            "滚".to_string(),
            "闭嘴".to_string(),
        ];
        
        // 正则表达式模式
        let patterns = vec![
            // 手机号码模式
            Regex::new(r"1[3-9]\d{9}").unwrap(),
            // 身份证号模式
            Regex::new(r"\d{15}|\d{18}").unwrap(),
            // 银行卡号模式
            Regex::new(r"\d{16,19}").unwrap(),
            // QQ号模式
            Regex::new(r"[1-9]\d{4,10}").unwrap(),
            // 微信号模式
            Regex::new(r"[a-zA-Z][a-zA-Z0-9_-]{5,19}").unwrap(),
            // 网址模式
            Regex::new(r"https?://[^\s]+").unwrap(),
            // 邮箱模式
            Regex::new(r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}").unwrap(),
        ];
        
        SensitiveWordConfig {
            critical_words,
            moderate_words,
            mild_words,
            patterns,
        }
    }
    
    /// 检查文本内容安全性
    pub fn check_text_content(&self, content: &str) -> SecurityCheckResult {
        debug!("检查文本内容安全性，长度: {}", content.len());
        
        let mut risk_score = 0.0;
        let mut detected_issues = Vec::new();
        let mut details = HashMap::new();
        
        // 1. 敏感词检测
        let sensitive_result = self.check_sensitive_words(content);
        risk_score += sensitive_result.0;
        detected_issues.extend(sensitive_result.1);
        
        // 2. 个人信息泄露检测
        let privacy_result = self.check_privacy_leak(content);
        risk_score += privacy_result.0;
        detected_issues.extend(privacy_result.1);
        
        // 3. 垃圾信息检测
        let spam_result = self.check_spam_content(content);
        risk_score += spam_result.0;
        detected_issues.extend(spam_result.1);
        
        // 4. 语言暴力检测
        let violence_result = self.check_language_violence(content);
        risk_score += violence_result.0;
        detected_issues.extend(violence_result.1);
        
        // 5. 内容质量评估
        let quality_result = self.assess_content_quality(content);
        risk_score += quality_result.0;
        detected_issues.extend(quality_result.1);
        
        // 记录检测详情
        details.insert("content_length".to_string(), serde_json::Value::Number(serde_json::Number::from(content.len())));
        details.insert("word_count".to_string(), serde_json::Value::Number(serde_json::Number::from(content.split_whitespace().count())));
        
        // 确定风险等级
        let risk_level = match risk_score {
            score if score >= 90.0 => SecurityRiskLevel::Critical,
            score if score >= 70.0 => SecurityRiskLevel::High,
            score if score >= 50.0 => SecurityRiskLevel::Medium,
            score if score >= 20.0 => SecurityRiskLevel::Low,
            _ => SecurityRiskLevel::Safe,
        };
        
        // 生成处理建议
        let action_recommendation = match risk_level {
            SecurityRiskLevel::Critical => "立即屏蔽内容，禁止发布".to_string(),
            SecurityRiskLevel::High => "屏蔽内容，限制用户发布权限".to_string(),
            SecurityRiskLevel::Medium => "内容需要人工审核后发布".to_string(),
            SecurityRiskLevel::Low => "内容可以发布，但降低推荐权重".to_string(),
            SecurityRiskLevel::Safe => "内容安全，正常发布".to_string(),
        };
        
        let requires_manual_review = matches!(risk_level, SecurityRiskLevel::Medium | SecurityRiskLevel::High);
        
        debug!("内容安全检测完成: 风险等级={:?}, 分数={:.2}", risk_level, risk_score);
        
        SecurityCheckResult {
            risk_level,
            risk_score,
            detected_issues,
            action_recommendation,
            requires_manual_review,
            details,
        }
    }
    
    /// 检查敏感词
    fn check_sensitive_words(&self, content: &str) -> (f64, Vec<String>) {
        let mut score = 0.0;
        let mut issues = Vec::new();
        let content_lower = content.to_lowercase();
        
        // 检查严重敏感词
        for word in &self.sensitive_words.critical_words {
            if content_lower.contains(&word.to_lowercase()) {
                score += 50.0; // 严重敏感词直接加50分
                issues.push(format!("包含严重敏感词: {}", word));
                warn!("检测到严重敏感词: {}", word);
            }
        }
        
        // 检查一般敏感词
        for word in &self.sensitive_words.moderate_words {
            if content_lower.contains(&word.to_lowercase()) {
                score += 20.0; // 一般敏感词加20分
                issues.push(format!("包含敏感词: {}", word));
            }
        }
        
        // 检查轻微敏感词
        for word in &self.sensitive_words.mild_words {
            if content_lower.contains(&word.to_lowercase()) {
                score += 5.0; // 轻微敏感词加5分
                issues.push(format!("包含不当词汇: {}", word));
            }
        }
        
        (score, issues)
    }
    
    /// 检查个人信息泄露
    fn check_privacy_leak(&self, content: &str) -> (f64, Vec<String>) {
        let mut score = 0.0;
        let mut issues = Vec::new();
        
        for pattern in &self.sensitive_words.patterns {
            if pattern.is_match(content) {
                score += 15.0;
                issues.push("可能包含个人敏感信息".to_string());
                break; // 避免重复计分
            }
        }
        
        (score, issues)
    }
    
    /// 检查垃圾信息
    fn check_spam_content(&self, content: &str) -> (f64, Vec<String>) {
        let mut score = 0.0;
        let mut issues = Vec::new();
        
        // 检查重复字符
        let repeated_char_ratio = self.calculate_repeated_char_ratio(content);
        if repeated_char_ratio > 0.3 {
            score += 20.0;
            issues.push("包含大量重复字符".to_string());
        }
        
        // 检查广告关键词
        let ad_keywords = ["加微信", "QQ群", "免费", "赚钱", "兼职", "代理", "优惠", "促销"];
        let mut ad_count = 0;
        for keyword in &ad_keywords {
            if content.contains(keyword) {
                ad_count += 1;
            }
        }
        
        if ad_count >= 3 {
            score += 25.0;
            issues.push("疑似广告或营销内容".to_string());
        } else if ad_count >= 1 {
            score += 10.0;
            issues.push("可能包含营销信息".to_string());
        }
        
        (score, issues)
    }
    
    /// 检查语言暴力
    fn check_language_violence(&self, content: &str) -> (f64, Vec<String>) {
        let mut score = 0.0;
        let mut issues = Vec::new();
        
        // 检查攻击性词汇
        let aggressive_words = ["滚", "死", "杀", "恨", "仇", "报复", "威胁"];
        let mut aggressive_count = 0;
        
        for word in &aggressive_words {
            if content.contains(word) {
                aggressive_count += 1;
            }
        }
        
        if aggressive_count >= 3 {
            score += 30.0;
            issues.push("包含严重攻击性语言".to_string());
        } else if aggressive_count >= 1 {
            score += 10.0;
            issues.push("包含攻击性词汇".to_string());
        }
        
        // 检查全大写文本（可能表示愤怒）
        let uppercase_ratio = content.chars().filter(|c| c.is_uppercase()).count() as f64 / content.len() as f64;
        if uppercase_ratio > 0.5 && content.len() > 10 {
            score += 5.0;
            issues.push("使用过多大写字母".to_string());
        }
        
        (score, issues)
    }
    
    /// 评估内容质量
    fn assess_content_quality(&self, content: &str) -> (f64, Vec<String>) {
        let mut score = 0.0;
        let mut issues = Vec::new();
        
        // 内容长度检查
        if content.len() < 5 {
            score += 10.0;
            issues.push("内容过短，可能为无意义内容".to_string());
        } else if content.len() > 5000 {
            score += 5.0;
            issues.push("内容过长".to_string());
        }
        
        // 特殊字符比例检查
        let special_char_count = content.chars().filter(|c| !c.is_alphanumeric() && !c.is_whitespace()).count();
        let special_char_ratio = special_char_count as f64 / content.len() as f64;
        
        if special_char_ratio > 0.3 {
            score += 15.0;
            issues.push("包含过多特殊字符".to_string());
        }
        
        // 数字比例检查（可能是垃圾信息）
        let digit_ratio = content.chars().filter(|c| c.is_numeric()).count() as f64 / content.len() as f64;
        if digit_ratio > 0.5 {
            score += 10.0;
            issues.push("包含过多数字".to_string());
        }
        
        (score, issues)
    }
    
    /// 计算重复字符比例
    fn calculate_repeated_char_ratio(&self, content: &str) -> f64 {
        if content.len() < 3 {
            return 0.0;
        }
        
        let mut repeated_count = 0;
        let chars: Vec<char> = content.chars().collect();
        
        for i in 0..chars.len() - 2 {
            if chars[i] == chars[i + 1] && chars[i + 1] == chars[i + 2] {
                repeated_count += 1;
            }
        }
        
        repeated_count as f64 / content.len() as f64
    }
    
    /// 检查图片内容（占位实现）
    pub fn check_image_content(&self, _image_url: &str) -> SecurityCheckResult {
        // TODO: 集成图像识别API (如腾讯云、阿里云的内容审核服务)
        debug!("图片内容安全检查 - 暂未实现");
        
        SecurityCheckResult {
            risk_level: SecurityRiskLevel::Safe,
            risk_score: 0.0,
            detected_issues: vec![],
            action_recommendation: "图片内容检查暂未实现，建议人工审核".to_string(),
            requires_manual_review: true,
            details: HashMap::new(),
        }
    }
    
    /// 检查用户资料安全性
    pub fn check_user_profile(&self, nickname: &str, bio: Option<&str>) -> SecurityCheckResult {
        debug!("检查用户资料安全性");
        
        let mut combined_content = nickname.to_string();
        if let Some(bio_text) = bio {
            combined_content.push_str(" ");
            combined_content.push_str(bio_text);
        }
        
        let mut result = self.check_text_content(&combined_content);
        
        // 用户资料的特殊检查
        if nickname.len() > 20 {
            result.risk_score += 5.0;
            result.detected_issues.push("昵称过长".to_string());
        }
        
        if nickname.chars().all(|c| c.is_numeric()) {
            result.risk_score += 10.0;
            result.detected_issues.push("昵称全为数字".to_string());
        }
        
        // 重新评估风险等级
        result.risk_level = match result.risk_score {
            score if score >= 90.0 => SecurityRiskLevel::Critical,
            score if score >= 70.0 => SecurityRiskLevel::High,
            score if score >= 50.0 => SecurityRiskLevel::Medium,
            score if score >= 20.0 => SecurityRiskLevel::Low,
            _ => SecurityRiskLevel::Safe,
        };
        
        result
    }
    
    /// 设置严格模式
    pub fn set_strict_mode(&mut self, enabled: bool) {
        self.strict_mode = enabled;
        debug!("内容安全严格模式: {}", if enabled { "启用" } else { "禁用" });
    }
    
    /// 获取内容安全统计信息
    pub fn get_security_stats(&self) -> HashMap<String, serde_json::Value> {
        let mut stats = HashMap::new();
        
        stats.insert("critical_words_count".to_string(), 
                    serde_json::Value::Number(serde_json::Number::from(self.sensitive_words.critical_words.len())));
        stats.insert("moderate_words_count".to_string(), 
                    serde_json::Value::Number(serde_json::Number::from(self.sensitive_words.moderate_words.len())));
        stats.insert("mild_words_count".to_string(), 
                    serde_json::Value::Number(serde_json::Number::from(self.sensitive_words.mild_words.len())));
        stats.insert("pattern_count".to_string(), 
                    serde_json::Value::Number(serde_json::Number::from(self.sensitive_words.patterns.len())));
        stats.insert("strict_mode".to_string(), 
                    serde_json::Value::Bool(self.strict_mode));
        
        stats
    }
}

impl Default for ContentSecurityService {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_safe_content() {
        let service = ContentSecurityService::new();
        let result = service.check_text_content("这是一条正常的消息内容");
        
        assert_eq!(result.risk_level, SecurityRiskLevel::Safe);
        assert!(result.risk_score < 20.0);
    }
    
    #[test]
    fn test_sensitive_word_detection() {
        let service = ContentSecurityService::new();
        let result = service.check_text_content("这里包含敏感词：杀人");
        
        assert!(result.risk_score > 0.0);
        assert!(!result.detected_issues.is_empty());
    }
    
    #[test]
    fn test_spam_detection() {
        let service = ContentSecurityService::new();
        let result = service.check_text_content("加微信免费赚钱兼职代理优惠促销");
        
        assert!(result.risk_score > 20.0);
        assert!(result.detected_issues.iter().any(|issue| issue.contains("营销")));
    }
    
    #[test]
    fn test_repeated_characters() {
        let service = ContentSecurityService::new();
        let result = service.check_text_content("哈哈哈哈哈哈哈哈哈哈哈哈哈哈哈");
        
        assert!(result.risk_score > 0.0);
        assert!(result.detected_issues.iter().any(|issue| issue.contains("重复字符")));
    }
    
    #[test]
    fn test_user_profile_check() {
        let service = ContentSecurityService::new();
        let result = service.check_user_profile("正常昵称", Some("这是一个正常的个人简介"));
        
        assert_eq!(result.risk_level, SecurityRiskLevel::Safe);
    }
}
