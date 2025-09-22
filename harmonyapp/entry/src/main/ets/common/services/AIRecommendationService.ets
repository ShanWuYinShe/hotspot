/**
 * HarmonyOS AI 智能推荐服务
 * 利用 HarmonyOS AI 能力提供个性化内容推荐
 */
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { LocationUtil } from '../location/LocationUtil';
import { StorageUtil } from '../utils/StorageUtil';

export interface RecommendationItem {
  id: string;
  type: 'moment' | 'group' | 'user' | 'location';
  title: string;
  description: string;
  score: number;
  reason: string;
  metadata: Record<string, any>;
}

export interface UserBehavior {
  userId: string;
  action: 'view' | 'like' | 'comment' | 'share' | 'join' | 'create';
  targetType: 'moment' | 'group' | 'user' | 'location';
  targetId: string;
  timestamp: number;
  location?: {
    latitude: number;
    longitude: number;
  };
  duration?: number; // 浏览时长（毫秒）
}

export interface UserPreferences {
  interests: string[]; // 兴趣标签
  locationPreference: 'nearby' | 'city' | 'global';
  contentTypes: string[]; // 偏好的内容类型
  activeTimeRanges: string[]; // 活跃时间段
  socialLevel: 'introvert' | 'moderate' | 'extrovert'; // 社交程度
}

export class AIRecommendationService {
  private static instance: AIRecommendationService;
  private userBehaviors: Map<string, UserBehavior[]> = new Map();
  private userPreferences: Map<string, UserPreferences> = new Map();
  private readonly maxBehaviorHistory = 1000;
  private readonly storageKey = 'ai_recommendation_data';

  private constructor() {
    this.loadFromStorage();
  }

  public static getInstance(): AIRecommendationService {
    if (!AIRecommendationService.instance) {
      AIRecommendationService.instance = new AIRecommendationService();
    }
    return AIRecommendationService.instance;
  }

  /**
   * 记录用户行为
   */
  public async recordUserBehavior(behavior: UserBehavior): Promise<void> {
    try {
      const userId = behavior.userId;
      
      if (!this.userBehaviors.has(userId)) {
        this.userBehaviors.set(userId, []);
      }
      
      const behaviors = this.userBehaviors.get(userId)!;
      behaviors.push(behavior);
      
      // 限制历史记录数量
      if (behaviors.length > this.maxBehaviorHistory) {
        behaviors.splice(0, behaviors.length - this.maxBehaviorHistory);
      }
      
      // 更新用户偏好
      await this.updateUserPreferences(userId, behavior);
      
      // 保存到本地存储
      await this.saveToStorage();
      
      hilog.info(0x0000, 'AIRecommendation', `记录用户行为: ${behavior.action} on ${behavior.targetType}`);
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'AIRecommendation', `记录用户行为失败: ${err.message}`);
    }
  }

  /**
   * 获取个性化推荐
   */
  public async getRecommendations(userId: string, count: number = 10): Promise<RecommendationItem[]> {
    try {
      const userPrefs = this.userPreferences.get(userId);
      const userBehaviors = this.userBehaviors.get(userId) || [];
      
      if (!userPrefs || userBehaviors.length === 0) {
        // 新用户或无行为数据，返回热门推荐
        return await this.getPopularRecommendations(count);
      }
      
      const recommendations: RecommendationItem[] = [];
      
      // 1. 基于兴趣的推荐
      const interestRecommendations = await this.getInterestBasedRecommendations(userId, userPrefs, Math.ceil(count * 0.4));
      recommendations.push(...interestRecommendations);
      
      // 2. 基于位置的推荐
      const locationRecommendations = await this.getLocationBasedRecommendations(userId, userPrefs, Math.ceil(count * 0.3));
      recommendations.push(...locationRecommendations);
      
      // 3. 基于社交关系的推荐
      const socialRecommendations = await this.getSocialBasedRecommendations(userId, userBehaviors, Math.ceil(count * 0.2));
      recommendations.push(...socialRecommendations);
      
      // 4. 探索性推荐（新内容）
      const explorationRecommendations = await this.getExplorationRecommendations(userId, userPrefs, Math.ceil(count * 0.1));
      recommendations.push(...explorationRecommendations);
      
      // 去重并按分数排序
      const uniqueRecommendations = this.deduplicateRecommendations(recommendations);
      return uniqueRecommendations.slice(0, count);
      
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'AIRecommendation', `获取推荐失败: ${err.message}`);
      return await this.getPopularRecommendations(count);
    }
  }

  /**
   * 基于兴趣的推荐
   */
  private async getInterestBasedRecommendations(userId: string, prefs: UserPreferences, count: number): Promise<RecommendationItem[]> {
    const recommendations: RecommendationItem[] = [];
    
    // 模拟基于兴趣标签的推荐算法
    for (const interest of prefs.interests.slice(0, count)) {
      recommendations.push({
        id: `interest_${interest}_${Date.now()}`,
        type: 'moment',
        title: `${interest}相关动态`,
        description: `发现更多关于${interest}的精彩内容`,
        score: 0.8 + Math.random() * 0.2,
        reason: `基于您对${interest}的兴趣`,
        metadata: { interest, algorithm: 'interest_based' }
      });
    }
    
    return recommendations;
  }

  /**
   * 基于位置的推荐
   */
  private async getLocationBasedRecommendations(userId: string, prefs: UserPreferences, count: number): Promise<RecommendationItem[]> {
    const recommendations: RecommendationItem[] = [];
    
    try {
      // 获取当前位置
      const currentLocation = await LocationUtil.getCurrentLocation();
      if (!currentLocation) {
        return recommendations;
      }
      
      // 根据位置偏好调整推荐范围
      let radius = 1000; // 默认1km
      switch (prefs.locationPreference) {
        case 'nearby':
          radius = 500;
          break;
        case 'city':
          radius = 10000;
          break;
        case 'global':
          radius = 100000;
          break;
      }
      
      // 模拟附近内容推荐
      for (let i = 0; i < count; i++) {
        recommendations.push({
          id: `location_${i}_${Date.now()}`,
          type: Math.random() > 0.5 ? 'moment' : 'group',
          title: `附近${radius < 1000 ? '500米' : radius < 10000 ? '1公里' : '城市'}内的内容`,
          description: `发现您周围的精彩动态和活跃群组`,
          score: 0.7 + Math.random() * 0.3,
          reason: `基于您的位置偏好`,
          metadata: { 
            location: currentLocation, 
            radius, 
            algorithm: 'location_based' 
          }
        });
      }
    } catch (error) {
      hilog.warn(0x0000, 'AIRecommendation', '获取位置推荐失败，跳过位置推荐');
    }
    
    return recommendations;
  }

  /**
   * 基于社交关系的推荐
   */
  private async getSocialBasedRecommendations(userId: string, behaviors: UserBehavior[], count: number): Promise<RecommendationItem[]> {
    const recommendations: RecommendationItem[] = [];
    
    // 分析用户的社交行为模式
    const socialInteractions = behaviors.filter(b => 
      ['like', 'comment', 'share', 'join'].includes(b.action)
    );
    
    // 找出经常互动的用户和群组
    const interactionTargets = new Map<string, number>();
    socialInteractions.forEach(interaction => {
      const key = `${interaction.targetType}_${interaction.targetId}`;
      interactionTargets.set(key, (interactionTargets.get(key) || 0) + 1);
    });
    
    // 基于互动频率生成推荐
    const sortedTargets = Array.from(interactionTargets.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, count);
    
    sortedTargets.forEach(([target, frequency], index) => {
      const [targetType, targetId] = target.split('_');
      recommendations.push({
        id: `social_${targetId}_${Date.now()}`,
        type: targetType as 'moment' | 'group' | 'user',
        title: `相关${targetType === 'user' ? '用户' : targetType === 'group' ? '群组' : '动态'}`,
        description: `基于您的社交互动推荐的内容`,
        score: 0.6 + (frequency / 10) * 0.4,
        reason: `基于您的社交互动模式`,
        metadata: { 
          targetId, 
          frequency, 
          algorithm: 'social_based' 
        }
      });
    });
    
    return recommendations;
  }

  /**
   * 探索性推荐（新内容）
   */
  private async getExplorationRecommendations(userId: string, prefs: UserPreferences, count: number): Promise<RecommendationItem[]> {
    const recommendations: RecommendationItem[] = [];
    
    // 推荐用户未接触过的内容类型
    const unexploredTypes = ['moment', 'group', 'user'];
    
    for (let i = 0; i < count && i < unexploredTypes.length; i++) {
      const type = unexploredTypes[i];
      recommendations.push({
        id: `exploration_${type}_${Date.now()}`,
        type: type as 'moment' | 'group' | 'user',
        title: `发现新的${type === 'moment' ? '动态' : type === 'group' ? '群组' : '用户'}`,
        description: `探索您可能感兴趣的新内容`,
        score: 0.5 + Math.random() * 0.3,
        reason: `为您推荐新的内容类型`,
        metadata: { 
          type, 
          algorithm: 'exploration' 
        }
      });
    }
    
    return recommendations;
  }

  /**
   * 获取热门推荐（新用户或无数据时使用）
   */
  private async getPopularRecommendations(count: number): Promise<RecommendationItem[]> {
    const recommendations: RecommendationItem[] = [];
    
    // 模拟热门内容推荐
    const popularTypes = ['moment', 'group', 'user'];
    
    for (let i = 0; i < count; i++) {
      const type = popularTypes[i % popularTypes.length];
      recommendations.push({
        id: `popular_${type}_${i}_${Date.now()}`,
        type: type as 'moment' | 'group' | 'user',
        title: `热门${type === 'moment' ? '动态' : type === 'group' ? '群组' : '用户'}`,
        description: `当前最受欢迎的内容`,
        score: 0.7 + Math.random() * 0.3,
        reason: `热门推荐`,
        metadata: { 
          type, 
          algorithm: 'popular' 
        }
      });
    }
    
    return recommendations;
  }

  /**
   * 更新用户偏好
   */
  private async updateUserPreferences(userId: string, behavior: UserBehavior): Promise<void> {
    let prefs = this.userPreferences.get(userId);
    
    if (!prefs) {
      prefs = {
        interests: [],
        locationPreference: 'nearby',
        contentTypes: [],
        activeTimeRanges: [],
        socialLevel: 'moderate'
      };
      this.userPreferences.set(userId, prefs);
    }
    
    // 根据行为更新偏好
    if (behavior.action === 'like' || behavior.action === 'share') {
      // 增加对该类型内容的偏好
      if (!prefs.contentTypes.includes(behavior.targetType)) {
        prefs.contentTypes.push(behavior.targetType);
      }
    }
    
    // 更新活跃时间
    const hour = new Date(behavior.timestamp).getHours();
    const timeRange = this.getTimeRange(hour);
    if (!prefs.activeTimeRanges.includes(timeRange)) {
      prefs.activeTimeRanges.push(timeRange);
    }
    
    // 更新社交程度
    if (['comment', 'share', 'join'].includes(behavior.action)) {
      if (prefs.socialLevel === 'introvert') {
        prefs.socialLevel = 'moderate';
      } else if (prefs.socialLevel === 'moderate') {
        prefs.socialLevel = 'extrovert';
      }
    }
  }

  /**
   * 获取时间段
   */
  private getTimeRange(hour: number): string {
    if (hour >= 6 && hour < 12) return 'morning';
    if (hour >= 12 && hour < 18) return 'afternoon';
    if (hour >= 18 && hour < 24) return 'evening';
    return 'night';
  }

  /**
   * 去重推荐结果
   */
  private deduplicateRecommendations(recommendations: RecommendationItem[]): RecommendationItem[] {
    const seen = new Set<string>();
    const unique: RecommendationItem[] = [];
    
    // 按分数排序
    recommendations.sort((a, b) => b.score - a.score);
    
    for (const item of recommendations) {
      const key = `${item.type}_${item.metadata.targetId || item.id}`;
      if (!seen.has(key)) {
        seen.add(key);
        unique.push(item);
      }
    }
    
    return unique;
  }

  /**
   * 从本地存储加载数据
   */
  private async loadFromStorage(): Promise<void> {
    try {
      const data = await StorageUtil.getItem(this.storageKey);
      if (data) {
        const parsed = JSON.parse(data);
        
        // 恢复用户行为数据
        if (parsed.userBehaviors) {
          this.userBehaviors = new Map(parsed.userBehaviors);
        }
        
        // 恢复用户偏好数据
        if (parsed.userPreferences) {
          this.userPreferences = new Map(parsed.userPreferences);
        }
        
        hilog.info(0x0000, 'AIRecommendation', '从本地存储加载推荐数据成功');
      }
    } catch (error) {
      hilog.warn(0x0000, 'AIRecommendation', '从本地存储加载推荐数据失败，使用默认设置');
    }
  }

  /**
   * 保存数据到本地存储
   */
  private async saveToStorage(): Promise<void> {
    try {
      const data = {
        userBehaviors: Array.from(this.userBehaviors.entries()),
        userPreferences: Array.from(this.userPreferences.entries())
      };
      
      await StorageUtil.setItem(this.storageKey, JSON.stringify(data));
    } catch (error) {
      hilog.error(0x0000, 'AIRecommendation', '保存推荐数据到本地存储失败');
    }
  }

  /**
   * 清理用户数据
   */
  public async clearUserData(userId: string): Promise<void> {
    this.userBehaviors.delete(userId);
    this.userPreferences.delete(userId);
    await this.saveToStorage();
    
    hilog.info(0x0000, 'AIRecommendation', `清理用户推荐数据: ${userId}`);
  }

  /**
   * 获取用户偏好分析
   */
  public getUserPreferencesAnalysis(userId: string): UserPreferences | null {
    return this.userPreferences.get(userId) || null;
  }
}
