import { FC, ReactNode } from 'react';

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
}

export const EmptyState: FC<EmptyStateProps> = ({
  title,
  description,
  icon,
  action,
}) => {
  return (
    <div
      role="region"
      aria-label={title}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '3rem 1.5rem',
        textAlign: 'center',
        backgroundColor: '#f8fafc',
        border: '1px dashed #cbd5e1',
        borderRadius: '0.75rem',
        color: '#475569',
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      {icon && <div style={{ marginBottom: '1rem', fontSize: '2rem' }}>{icon}</div>}
      <h3 style={{ margin: '0 0 0.5rem 0', fontSize: '1.125rem', color: '#1e293b' }}>{title}</h3>
      {description && (
        <p style={{ margin: '0 0 1.5rem 0', fontSize: '0.875rem', color: '#64748b', maxWidth: '400px' }}>
          {description}
        </p>
      )}
      {action && <div>{action}</div>}
    </div>
  );
};
