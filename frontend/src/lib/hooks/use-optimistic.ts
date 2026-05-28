/**
 * 通用乐观更新 Toggle Hook (React)
 */

import { useState, useRef, useCallback, useEffect } from "react";
import { toast } from "sonner";
import { isSuccess, type ApiResponse } from "@/lib/api/client";

interface OptimisticOptions<T> {
  initialValue: boolean | (() => boolean);
  initialCount?: number | (() => number);
  onTrue: () => Promise<ApiResponse<T>>;
  onFalse: () => Promise<ApiResponse<T>>;
  errorMessage?: string;
}

export function useOptimisticToggle<T>(options: OptimisticOptions<T>) {
  const getExternalValue = () =>
    typeof options.initialValue === "function"
      ? options.initialValue()
      : options.initialValue;

  const getExternalCount = () =>
    options.initialCount !== undefined
      ? typeof options.initialCount === "function"
        ? options.initialCount()
        : options.initialCount
      : 0;

  const [isOn, setIsOn] = useState(getExternalValue);
  const [count, setCount] = useState(getExternalCount);
  const [isLoading, setIsLoading] = useState(false);
  const aliveRef = useRef(true);

  const prevExternalValueRef = useRef(getExternalValue());
  const prevExternalCountRef = useRef(getExternalCount());

  // 同步外部值变化（避免乐观更新后被旧的外部值覆盖）
  useEffect(() => {
    const extVal = getExternalValue();
    const extCount = getExternalCount();

    if (extVal !== prevExternalValueRef.current) {
      prevExternalValueRef.current = extVal;
      setIsOn(extVal);
    }
    if (extCount !== prevExternalCountRef.current) {
      prevExternalCountRef.current = extCount;
      setCount(extCount);
    }
  });

  useEffect(() => {
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const toggle = useCallback(async () => {
    if (isLoading) return;

    const previousIsOn = isOn;
    const previousCount = count;

    const nextIsOn = !previousIsOn;
    setIsOn(nextIsOn);
    if (options.initialCount !== undefined) {
      setCount((c) => c + (nextIsOn ? 1 : -1));
    }

    setIsLoading(true);

    try {
      const action = nextIsOn ? options.onTrue : options.onFalse;
      const response = await action();

      if (!isSuccess(response)) {
        throw new Error(response.message || "Operation failed");
      }
    } catch (error) {
      if (!aliveRef.current) return;
      setIsOn(previousIsOn);
      setCount(previousCount);
      toast.error(
        options.errorMessage ||
          (error instanceof Error ? error.message : "操作失败"),
      );
    } finally {
      if (aliveRef.current) setIsLoading(false);
    }
  }, [isLoading, isOn, count, options]);

  return { isOn, count, isLoading, toggle };
}
