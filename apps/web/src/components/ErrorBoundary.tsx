import { Component, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}
interface State {
  crashed: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { crashed: false };

  static getDerivedStateFromError(): State {
    return { crashed: true };
  }

  render() {
    if (!this.state.crashed) return this.props.children;
    return (
      <main className="splash">
        <div className="empty-hint">
          <strong>Something went wrong</strong>
          <span>Reload the page to recover. Your data is safe on the server.</span>
          <button type="button" className="btn btn-primary" onClick={() => window.location.reload()}>
            Reload
          </button>
        </div>
      </main>
    );
  }
}
