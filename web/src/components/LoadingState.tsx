import { FC } from 'react';

interface LoadingStateProps {
  message?: string;
  size?: 'small' | 'medium' | 'large';
}

export const LoadingState: FC<LoadingStateProps> = ({
  message = 'Loading...',
  size = 'medium',
}) => {
  const spinnerSize = size === 'small' ? '16px' : size === 'large' ? '40px' : '24px';

  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '2rem',
        color: '#64748b',
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      <div
        style={{
          width: spinnerSize,
          height: spinnerSize,
          border: '3px solid #e2e8f0',
          borderTopColor: '#3b82f6',
          borderRadius: '50%',
          animation: 'spin 1s linear infinite',
          marginBottom: message ? '0.75rem' : 0,
        }}
      />
      <style>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
      {message && <span style={{ fontSize: '0.875rem' }}>{message}</span>}
    </div>
  );
};
