import { FC } from 'react';

interface SkeletonProps {
  height?: string;
  width?: string;
  borderRadius?: string;
  count?: number;
}

export const Skeleton: FC<SkeletonProps> = ({
  height = '1.25rem',
  width = '100%',
  borderRadius = '0px',
  count = 1,
}) => {
  return (
    <div data-testid="loading-skeleton" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem', width: '100%' }}>
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          style={{
            height,
            width,
            borderRadius,
            backgroundColor: '#e2e8f0',
            animation: 'pulse 1.5s ease-in-out infinite',
          }}
        />
      ))}
      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
      `}</style>
    </div>
  );
};
