import { EdgeProps, getBezierPath, EdgeLabelRenderer, BaseEdge } from '@xyflow/react';

export function LabeledEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  markerEnd,
  label,
  data,
  selected,
}: EdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const displayLabel =
    (label as string) ||
    (data?.label as string) ||
    (data?.condition as string) ||
    '';

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: selected ? '#c0c1ff' : '#464554',
          strokeWidth: selected ? 2.5 : 2,
          strokeDasharray: selected ? undefined : '6 4',
          filter: selected ? 'drop-shadow(0 0 6px rgba(192, 193, 255, 0.4))' : undefined,
          ...style,
        }}
        markerEnd={markerEnd}
      />
      {displayLabel && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: 'all',
            }}
            className={`nodrag nopan px-2.5 py-1 rounded-full text-[10px] font-mono font-semibold transition-all backdrop-blur-md border ${
              selected
                ? 'bg-[#b76dff]/30 text-[#ddb7ff] border-[#ddb7ff] shadow-[0_0_12px_rgba(221,183,255,0.4)]'
                : 'bg-[#171b26]/90 text-[#c7c4d7] border-[#464554] shadow-md'
            }`}
            data-testid={`edge-label-${id}`}
          >
            {displayLabel}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}
