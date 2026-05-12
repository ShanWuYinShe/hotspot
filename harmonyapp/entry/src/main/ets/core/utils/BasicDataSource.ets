/**
 * 通用 BasicDataSource，用于 LazyForEach
 */
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF00;
const TAG: string = 'BasicDataSource';

export class BasicDataSource<T> implements IDataSource {
  private listeners: DataChangeListener[] = [];
  private originDataArray: T[] = [];

  constructor(data: T[]) {
    this.originDataArray = data;
  }

  public totalCount(): number {
    return this.originDataArray.length;
  }

  public getData(index: number): T {
    return this.originDataArray[index];
  }

  // 辅助方法：获取数据在数组中的索引
  public getDataIndex(data: T): number {
    return this.originDataArray.indexOf(data);
  }

  // 该方法为框架侧调用，为LazyForEach组件向其数据源处添加listener监听
  registerDataChangeListener(listener: DataChangeListener): void {
    if (this.listeners.indexOf(listener) < 0) {
      hilog.debug(DOMAIN_NUMBER, TAG, 'add listener');
      this.listeners.push(listener);
    }
  }

  // 该方法为框架侧调用，为对应的LazyForEach组件在数据源处移除listener监听
  unregisterDataChangeListener(listener: DataChangeListener): void {
    const pos = this.listeners.indexOf(listener);
    if (pos >= 0) {
      hilog.debug(DOMAIN_NUMBER, TAG, 'remove listener');
      this.listeners.splice(pos, 1);
    }
  }

  // 通知LazyForEach组件需要重载所有子组件
  notifyDataReload(): void {
    this.listeners.forEach(listener => {
      listener.onDataReloaded();
    })
  }

  // 通知LazyForEach组件需要在index对应索引处添加子组件
  notifyDataAdd(index: number): void {
    this.listeners.forEach(listener => {
      listener.onDataAdd(index);
    })
  }

  // 通知LazyForEach组件在index对应索引处数据有变化，需要重建该子组件
  notifyDataChange(index: number): void {
    this.listeners.forEach(listener => {
      listener.onDataChange(index);
    })
  }

  // 通知LazyForEach组件需要在index对应索引处删除该子组件
  notifyDataDelete(index: number): void {
    this.listeners.forEach(listener => {
      listener.onDataDelete(index);
    })
  }

  // 通知LazyForEach组件将from索引处的子组件移动到to索引处
  notifyDataMove(from: number, to: number): void {
    this.listeners.forEach(listener => {
      listener.onDataMove(from, to);
    })
  }

  // 业务方法
  public setData(data: T[]) {
    this.originDataArray = data;
    this.notifyDataReload();
  }

  public addData(data: T[]) {
    if (!data || data.length === 0) return;
    const startIndex = this.originDataArray.length;
    for (const item of data) {
      this.originDataArray.push(item);
    }
    for (let i = startIndex; i < this.originDataArray.length; i++) {
      this.notifyDataAdd(i);
    }
  }

  /** 追加数据到末尾（别名） */
  public appendData(data: T[]) {
    this.addData(data);
  }

  /** 在头部插入数据 */
  public prependData(data: T[]) {
    if (!data || data.length === 0) return;
    const newArr: T[] = [];
    for (const item of data) {
      newArr.push(item);
    }
    for (const item of this.originDataArray) {
      newArr.push(item);
    }
    this.originDataArray = newArr;
    this.notifyDataReload();
  }
  
  public deleteData(index: number) {
    if (index >= 0 && index < this.originDataArray.length) {
      this.originDataArray.splice(index, 1);
      this.notifyDataDelete(index);
    }
  }
  
  public clear() {
      this.originDataArray = [];
      this.notifyDataReload();
  }
}
