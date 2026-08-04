import { useState, useCallback } from 'react';

export type AnnouncePoliteness = 'polite' | 'assertive';

export interface Announcement {
  id: number;
  message: string;
  politeness: AnnouncePoliteness;
}

export function useAnnouncer() {
  const [announcement, setAnnouncement] = useState<Announcement | null>(null);

  const announce = useCallback((message: string, politeness: AnnouncePoliteness = 'polite') => {
    setAnnouncement({
      id: Date.now(),
      message,
      politeness,
    });
  }, []);

  const clearAnnouncement = useCallback(() => {
    setAnnouncement(null);
  }, []);

  return {
    announcement,
    announce,
    clearAnnouncement,
  };
}

export function LiveAnnouncer({ announcement }: { announcement: Announcement | null }) {
  if (!announcement) return null;

  return (
    <div
      role={announcement.politeness === 'assertive' ? 'alert' : 'status'}
      aria-live={announcement.politeness}
      aria-atomic="true"
      style={{
        position: 'absolute',
        width: '1px',
        height: '1px',
        padding: 0,
        margin: '-1px',
        overflow: 'hidden',
        clip: 'rect(0, 0, 0, 0)',
        whiteSpace: 'nowrap',
        border: 0,
      }}
      data-testid="live-announcer"
    >
      {announcement.message}
    </div>
  );
}
