import { Component, ReactNode, ErrorInfo } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode | ((error: Error, reset: () => void) => ReactNode);
  onReset?: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
    };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    if (typeof window !== 'undefined') {
      (window as any).__LAST_REACT_ERROR__ = {
        message: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
      };
    }
  }

  resetBoundary = (): void => {
    this.props.onReset?.();
    this.setState({
      hasError: false,
      error: null,
    });
  };

  render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, fallback } = this.props;

    if (hasError && error) {
      if (typeof fallback === 'function') {
        return fallback(error, this.resetBoundary);
      }
      if (fallback) {
        return fallback;
      }
      return (
        <div
          role="alert"
          style={{
            padding: '2rem',
            margin: '1rem',
            border: '1px solid #ef4444',
            borderRadius: '0px',
            backgroundColor: '#fef2f2',
            color: '#991b1b',
            fontFamily: 'system-ui, sans-serif',
          }}
        >
          <h2 style={{ marginTop: 0, fontSize: '1.25rem' }}>Something went wrong</h2>
          <p style={{ fontSize: '0.875rem' }}>{error.message || 'An unexpected runtime error occurred.'}</p>
          <button
            onClick={this.resetBoundary}
            style={{
              padding: '0.5rem 1rem',
              backgroundColor: '#dc2626',
              color: '#ffffff',
              border: 'none',
              borderRadius: '0px',
              cursor: 'pointer',
              fontWeight: 500,
            }}
          >
            Try Again
          </button>
        </div>
      );
    }

    return children;
  }
}
