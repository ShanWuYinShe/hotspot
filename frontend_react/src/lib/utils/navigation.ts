import type { NavigateFunction } from "react-router";

export function goBack(navigate: NavigateFunction, fallback = "/") {
  if (window.history.length > 1) {
    navigate(-1);
  } else {
    navigate(fallback, { replace: true });
  }
}
