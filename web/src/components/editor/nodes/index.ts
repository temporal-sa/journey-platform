import type { NodeTypes } from '@xyflow/react';
import { EventStartNode } from './EventStartNode';
import { ConditionNode } from './ConditionNode';
import { DelayNode } from './DelayNode';
import { WaitForEventNode } from './WaitForEventNode';
import { ExperimentNode } from './ExperimentNode';
import { EmailNode } from './EmailNode';
import { SMSNode } from './SMSNode';
import { PushNode } from './PushNode';
import { InAppNode } from './InAppNode';
import { WebhookNode } from './WebhookNode';
import { ExitNode } from './ExitNode';

export {
  EventStartNode,
  ConditionNode,
  DelayNode,
  WaitForEventNode,
  ExperimentNode,
  EmailNode,
  SMSNode,
  PushNode,
  InAppNode,
  WebhookNode,
  ExitNode,
};

/* eslint-disable @typescript-eslint/no-explicit-any */
export const nodeTypes: NodeTypes = {
  EventStart: EventStartNode as any,
  EventStartNode: EventStartNode as any,
  trigger: EventStartNode as any,
  eventstart: EventStartNode as any,

  Condition: ConditionNode as any,
  ConditionNode: ConditionNode as any,
  condition: ConditionNode as any,

  Delay: DelayNode as any,
  DelayNode: DelayNode as any,
  delay: DelayNode as any,

  WaitForEvent: WaitForEventNode as any,
  WaitForEventNode: WaitForEventNode as any,
  wait_for_event: WaitForEventNode as any,
  waitforevent: WaitForEventNode as any,

  Experiment: ExperimentNode as any,
  ExperimentNode: ExperimentNode as any,
  experiment: ExperimentNode as any,

  Email: EmailNode as any,
  EmailNode: EmailNode as any,
  email: EmailNode as any,
  action: EmailNode as any,

  SMS: SMSNode as any,
  SMSNode: SMSNode as any,
  sms: SMSNode as any,

  Push: PushNode as any,
  PushNode: PushNode as any,
  push: PushNode as any,

  InApp: InAppNode as any,
  InAppNode: InAppNode as any,
  inapp: InAppNode as any,

  Webhook: WebhookNode as any,
  WebhookNode: WebhookNode as any,
  webhook: WebhookNode as any,

  Exit: ExitNode as any,
  ExitNode: ExitNode as any,
  exit: ExitNode as any,
};
