import { Component } from "react";
import { AlertTriangle, RotateCcw } from "lucide-react";
import { motion } from "motion/react";

interface Props {
  children: React.ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  retryKey: number;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null, retryKey: 0 };

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  handleRetry = () => {
    this.setState((prev) => ({
      hasError: false,
      error: null,
      retryKey: prev.retryKey + 1,
    }));
  };

  render() {
    if (this.state.hasError) {
      return (
        <div
          className="flex flex-col items-center justify-center gap-6"
          style={{
            height: "100%",
            backgroundColor: "var(--neu-bg)",
            padding: 24,
          }}
        >
          <div
            className="flex items-center justify-center rounded-full"
            style={{
              width: 56,
              height: 56,
              background: "var(--neu-accent-soft)",
            }}
          >
            <AlertTriangle
              style={{
                width: 24,
                height: 24,
                color: "var(--neu-accent-primary)",
              }}
              strokeWidth={1.5}
            />
          </div>
          <div className="flex flex-col items-center gap-2">
            <p
              style={{
                fontSize: 17,
                color: "var(--neu-text)",
                fontWeight: 500,
              }}
            >
              出了点问题
            </p>
            <p
              style={{
                fontSize: 13,
                color: "var(--neu-text-muted)",
                textAlign: "center",
                maxWidth: 280,
              }}
            >
              页面遇到了意外错误，请尝试刷新
            </p>
          </div>
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={this.handleRetry}
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-full outline-none"
            style={{
              background: "var(--neu-accent-primary)",
              color: "white",
              fontSize: 13,
              boxShadow: "0 4px 14px var(--neu-glow)",
            }}
          >
            <RotateCcw style={{ width: 14, height: 14 }} strokeWidth={1.5} />
            重试
          </motion.button>
        </div>
      );
    }

    return <div key={this.state.retryKey}>{this.props.children}</div>;
  }
}
