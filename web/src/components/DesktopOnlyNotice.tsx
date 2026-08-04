import { useState, useEffect } from 'react';

export interface DesktopOnlyNoticeProps {
  minWidth?: number; // default 1024
  onNavigateToList?: () => void;
  onDismissOverride?: () => void;
}

export function DesktopOnlyNotice({
  minWidth = 1024,
  onNavigateToList,
  onDismissOverride,
}: DesktopOnlyNoticeProps) {
  const [windowWidth, setWindowWidth] = useState<number>(
    typeof window !== 'undefined' ? window.innerWidth : 1200
  );
  const [isDismissed, setIsDismissed] = useState(false);

  useEffect(() => {
    const handleResize = () => {
      setWindowWidth(window.innerWidth);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  if (windowWidth >= minWidth || isDismissed) {
    return null;
  }

  return (
    <div
      role="region"
      aria-label="Desktop Display Notice"
      aria-labelledby="desktop-notice-title"
      aria-describedby="desktop-notice-desc"
      data-testid="desktop-only-notice"
      style={{
        position: 'absolute',
        inset: 0,
        backgroundColor: 'rgba(15, 23, 42, 0.92)',
        backdropFilter: 'blur(6px)',
        zIndex: 500,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '2rem',
        color: '#ffffff',
        textAlign: 'center',
      }}
    >
      <div
        style={{
          maxWidth: '480px',
          backgroundColor: '#1e293b',
          border: '1px solid #334155',
          borderRadius: '0px',
          padding: '2rem',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
        }}
      >
        <div style={{ fontSize: '3rem', marginBottom: '1rem' }} aria-hidden="true">
          💻
        </div>
        <h2
          id="desktop-notice-title"
          style={{ margin: '0 0 0.75rem 0', fontSize: '1.25rem', fontWeight: 700, color: '#f8fafc' }}
        >
          Desktop Display Required for Canvas Editor
        </h2>
        <p
          id="desktop-notice-desc"
          style={{ margin: 0, fontSize: '0.875rem', color: '#94a3b8', lineHeight: 1.5 }}
        >
          The Journey Graph Canvas requires a screen width of at least {minWidth}px for full node drag-and-drop, wiring handles, and multi-tab inspecting (current width: {windowWidth}px).
        </p>

        <div
          style={{
            marginTop: '1.5rem',
            display: 'flex',
            flexDirection: 'column',
            gap: '0.75rem',
          }}
        >
          {onNavigateToList && (
            <button
              onClick={onNavigateToList}
              data-testid="desktop-notice-nav-btn"
              style={{
                padding: '0.625rem 1.25rem',
                backgroundColor: '#2563eb',
                color: '#ffffff',
                border: 'none',
                borderRadius: '0px',
                fontWeight: 600,
                fontSize: '0.875rem',
                cursor: 'pointer',
              }}
            >
              Return to Journeys List
            </button>
          )}

          <button
            onClick={() => {
              setIsDismissed(true);
              onDismissOverride?.();
            }}
            data-testid="desktop-notice-dismiss-btn"
            style={{
              padding: '0.5rem 1rem',
              backgroundColor: 'transparent',
              color: '#94a3b8',
              border: '1px solid #475569',
              borderRadius: '0px',
              fontWeight: 500,
              fontSize: '0.8125rem',
              cursor: 'pointer',
            }}
          >
            Continue to Canvas Anyway
          </button>
        </div>
      </div>
    </div>
  );
}
