import React, { useState, useEffect, useMemo } from 'react';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { JourneyApiClient } from '../../api/client';
import type { CatalogRecord } from '../../types/api';

const apiClient = new JourneyApiClient();

export type CatalogComponentType = 'template' | 'parameter' | 'attribute' | 'event' | 'action' | 'metric';

export interface CatalogCreationModalProps {
  isOpen: boolean;
  onClose: () => void;
  defaultType?: string;
  onSuccess?: (createdRecord: CatalogRecord) => void;
}

const mapDefaultTypeToComponentType = (type?: string): CatalogComponentType => {
  if (!type) return 'template';
  const lower = type.toLowerCase();
  if (lower.includes('template')) return 'template';
  if (lower.includes('param')) return 'parameter';
  if (lower.includes('attr')) return 'attribute';
  if (lower.includes('event') || lower.includes('trigger')) return 'event';
  if (lower.includes('action') || lower.includes('activity')) return 'action';
  if (lower.includes('metric')) return 'metric';
  return 'template';
};

const mapComponentTypeToApiPath = (compType: CatalogComponentType): string => {
  switch (compType) {
    case 'template':
      return 'templates';
    case 'parameter':
      return 'parameters';
    case 'attribute':
      return 'attributes';
    case 'event':
      return 'events';
    case 'action':
      return 'actions';
    case 'metric':
      return 'metrics';
    default:
      return 'templates';
  }
};

export const CatalogCreationModal: React.FC<CatalogCreationModalProps> = ({
  isOpen,
  onClose,
  defaultType,
  onSuccess,
}) => {
  const [componentType, setComponentType] = useState<CatalogComponentType>('template');
  const [name, setName] = useState('');
  const [recordId, setRecordId] = useState('');
  const [version, setVersion] = useState('1.0.0');
  const [description, setDescription] = useState('');
  const [tagsInput, setTagsInput] = useState('');

  // Template specific state
  const [templateChannel, setTemplateChannel] = useState<'email' | 'sms' | 'push' | 'webhook'>('email');
  const [templateSubject, setTemplateSubject] = useState('');
  const [templateBody, setTemplateBody] = useState('');

  // Parameter specific state
  const [paramDataType, setParamDataType] = useState<'string' | 'number' | 'boolean' | 'json' | 'duration'>('string');
  const [paramDefaultValue, setParamDefaultValue] = useState('');
  const [paramScope, setParamScope] = useState<'execution' | 'config' | 'global'>('execution');

  // Attribute specific state
  const [attrDataType, setAttrDataType] = useState<'string' | 'number' | 'boolean' | 'array' | 'object'>('string');
  const [attrIsPii, setAttrIsPii] = useState(false);
  const [attrDefaultValue, setAttrDefaultValue] = useState('');

  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      const initialType = mapDefaultTypeToComponentType(defaultType);
      setComponentType(initialType);
      setName('');
      setRecordId('');
      setVersion('1.0.0');
      setDescription('');
      setTagsInput('');
      setTemplateChannel('email');
      setTemplateSubject('');
      setTemplateBody('');
      setParamDataType('string');
      setParamDefaultValue('');
      setParamScope('execution');
      setAttrDataType('string');
      setAttrIsPii(false);
      setAttrDefaultValue('');
      setErrorMessage(null);
      setIsSaving(false);
    }
  }, [isOpen, defaultType]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setErrorMessage('Component name is required.');
      return;
    }

    setIsSaving(true);
    setErrorMessage(null);

    try {
      const parsedTags = tagsInput
        .split(',')
        .map((t) => t.trim().toLowerCase())
        .filter(Boolean);

      if (!parsedTags.includes(componentType)) {
        parsedTags.push(componentType);
      }

      const schemaDefinition: Record<string, unknown> = {};

      if (componentType === 'template') {
        schemaDefinition.subject = templateSubject.trim();
        schemaDefinition.body = templateBody.trim();
        schemaDefinition.channel = templateChannel;
      } else if (componentType === 'parameter') {
        schemaDefinition.data_type = paramDataType;
        schemaDefinition.default_value = paramDefaultValue.trim();
        schemaDefinition.scope = paramScope;
      } else if (componentType === 'attribute') {
        schemaDefinition.data_type = attrDataType;
        schemaDefinition.is_pii = attrIsPii;
        schemaDefinition.default_value = attrDefaultValue.trim();
      }

      const baseSlug = name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '');
      const uniqueSuffix = Math.random().toString(36).substring(2, 8);
      const generatedId =
        recordId.trim() ||
        `cat-${componentType}-${baseSlug || 'item'}-${uniqueSuffix}`;

      const payload: Partial<CatalogRecord> = {
        schema_version: '1.0',
        record_id: generatedId,
        name: name.trim(),
        component_type: (componentType === 'event' ? 'trigger' : componentType === 'attribute' ? 'condition' : componentType) as CatalogRecord['component_type'],
        version: version.trim() || '1.0.0',
        description: description.trim(),
        schema_definition: schemaDefinition,
        tags: parsedTags,
        is_deprecated: false,
      };

      const apiPath = mapComponentTypeToApiPath(componentType);
      const createdRecord = await apiClient.createCatalogItem(apiPath, payload);

      if (onSuccess) {
        onSuccess(createdRecord);
      }
      onClose();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to create catalog component. Please check your inputs.';
      setErrorMessage(msg);
    } finally {
      setIsSaving(false);
    }
  };

  const footer = (
    <div className="flex items-center justify-end gap-3 w-full border-t border-[#464554] pt-4 mt-2">
      <Button
        type="button"
        variant="secondary-dark"
        onClick={onClose}
        disabled={isSaving}
        data-testid="catalog-create-cancel-btn"
      >
        Cancel
      </Button>
      <Button
        type="button"
        variant="primary-purple"
        icon="add_circle"
        onClick={handleSubmit}
        isLoading={isSaving}
        disabled={isSaving}
        data-testid="catalog-create-submit-btn"
        className="font-bold cursor-pointer"
      >
        Create Component
      </Button>
    </div>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Create Catalog Component"
      subtitle="Define new journey schema components including templates, parameters, and attributes."
      icon="add_box"
      iconAccentColor="#ddb7ff"
      maxWidth="2xl"
      footer={footer}
      testId="catalog-creation-modal"
    >
      <form onSubmit={handleSubmit} className="space-y-4" data-testid="catalog-creation-form">
        {errorMessage && (
          <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs rounded-none flex items-center gap-2">
            <span className="material-symbols-outlined text-base">error</span>
            <span>{errorMessage}</span>
          </div>
        )}

        {/* Component Type Selector */}
        <div>
          <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
            Component Type <span className="text-rose-400">*</span>
          </label>
          <select
            value={componentType}
            onChange={(e) => setComponentType(e.target.value as CatalogComponentType)}
            data-testid="catalog-type-select"
            className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
          >
            <option value="template">Template Component (Message/Email Content)</option>
            <option value="parameter">Parameter Component (Execution Parameter)</option>
            <option value="attribute">Subject Attribute (User/Subject Property)</option>
            <option value="event">Event Component (Trigger Schema)</option>
            <option value="action">Action Component (Side-effect Activity)</option>
            <option value="metric">Metric Component (Analytics Goal)</option>
          </select>
        </div>

        {/* Basic Metadata Section */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
              Component Name <span className="text-rose-400">*</span>
            </label>
            <input
              type="text"
              placeholder="e.g. Order Confirmation Email"
              value={name}
              onChange={(e) => setName(e.target.value)}
              data-testid="catalog-name-input"
              required
              className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
              Record ID / Key <span className="text-[#908fa0] font-normal">(Auto-generated if empty)</span>
            </label>
            <input
              type="text"
              placeholder="e.g. cat-tmpl-order-confirm"
              value={recordId}
              onChange={(e) => setRecordId(e.target.value)}
              data-testid="catalog-record-id-input"
              className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
              Version
            </label>
            <input
              type="text"
              placeholder="1.0.0"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              data-testid="catalog-version-input"
              className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
              Tags <span className="text-[#908fa0] font-normal">(Comma separated)</span>
            </label>
            <input
              type="text"
              placeholder="email, onboarding, notification"
              value={tagsInput}
              onChange={(e) => setTagsInput(e.target.value)}
              data-testid="catalog-tags-input"
              className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>
        </div>

        <div>
          <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
            Description
          </label>
          <textarea
            rows={2}
            placeholder="Detailed description of component purpose and usage..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            data-testid="catalog-description-input"
            className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
          />
        </div>

        {/* Component Type Specific Settings */}
        <div className="border-t border-[#464554] pt-4 mt-2">
          <h4 className="text-xs font-bold text-[#ddb7ff] uppercase tracking-wider mb-3 flex items-center gap-1.5 font-['Outfit']">
            <span className="material-symbols-outlined text-sm">tune</span>
            <span>{componentType.toUpperCase()} Specific Schema Configuration</span>
          </h4>

          {componentType === 'template' && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Delivery Channel
                </label>
                <select
                  value={templateChannel}
                  onChange={(e) => setTemplateChannel(e.target.value as 'email' | 'sms' | 'push' | 'webhook')}
                  data-testid="catalog-template-channel-select"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                >
                  <option value="email">Email</option>
                  <option value="sms">SMS Text Message</option>
                  <option value="push">Push Notification</option>
                  <option value="webhook">Webhook Payload</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Subject Line / Header
                </label>
                <input
                  type="text"
                  placeholder="Welcome {{subject.first_name}} to {{parameter.company_name}}!"
                  value={templateSubject}
                  onChange={(e) => setTemplateSubject(e.target.value)}
                  data-testid="catalog-template-subject-input"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Template Content Body
                </label>
                <textarea
                  rows={4}
                  placeholder="Hi {{subject.first_name}},\n\nWelcome to {{parameter.product_name}}! We are excited to have you on board.\n\nBest regards,\nThe Team"
                  value={templateBody}
                  onChange={(e) => setTemplateBody(e.target.value)}
                  data-testid="catalog-template-body-input"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-mono text-xs"
                />
              </div>
            </div>
          )}

          {componentType === 'parameter' && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Data Type
                </label>
                <select
                  value={paramDataType}
                  onChange={(e) => setParamDataType(e.target.value as 'string' | 'number' | 'boolean' | 'json' | 'duration')}
                  data-testid="catalog-param-datatype-select"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                >
                  <option value="string">String</option>
                  <option value="number">Number</option>
                  <option value="boolean">Boolean</option>
                  <option value="json">JSON Object</option>
                  <option value="duration">Duration (Seconds)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Default Value
                </label>
                <input
                  type="text"
                  placeholder="e.g. 30 or true or support@example.com"
                  value={paramDefaultValue}
                  onChange={(e) => setParamDefaultValue(e.target.value)}
                  data-testid="catalog-param-default-input"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                  Parameter Scope
                </label>
                <select
                  value={paramScope}
                  onChange={(e) => setParamScope(e.target.value as 'execution' | 'config' | 'global')}
                  data-testid="catalog-param-scope-select"
                  className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                >
                  <option value="execution">Execution Scope</option>
                  <option value="config">Config Scope</option>
                  <option value="global">Global Scope</option>
                </select>
              </div>
            </div>
          )}

          {componentType === 'attribute' && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                    Attribute Data Type
                  </label>
                  <select
                    value={attrDataType}
                    onChange={(e) => setAttrDataType(e.target.value as 'string' | 'number' | 'boolean' | 'array' | 'object')}
                    data-testid="catalog-attr-datatype-select"
                    className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                  >
                    <option value="string">String</option>
                    <option value="number">Number</option>
                    <option value="boolean">Boolean</option>
                    <option value="array">Array</option>
                    <option value="object">Object</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-medium text-[#c7c4d7] mb-1 font-['Outfit']">
                    Default / Example Value
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. gold or 500"
                    value={attrDefaultValue}
                    onChange={(e) => setAttrDefaultValue(e.target.value)}
                    data-testid="catalog-attr-default-input"
                    className="w-full px-3 py-2 bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs rounded-none focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
                  />
                </div>
              </div>

              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id="attrIsPii"
                  checked={attrIsPii}
                  onChange={(e) => setAttrIsPii(e.target.checked)}
                  data-testid="catalog-attr-pii-checkbox"
                  className="h-4 w-4 rounded-none border-[#464554] bg-[#171b26] text-[#b76dff] focus:ring-[#c0c1ff]"
                />
                <label htmlFor="attrIsPii" className="text-xs text-[#dfe2f1] font-['Outfit'] cursor-pointer">
                  Contains PII / Sensitive Subject Data
                </label>
              </div>
            </div>
          )}

          {(componentType === 'event' || componentType === 'action' || componentType === 'metric') && (
            <div className="p-3 bg-[#171b26] border border-[#464554] text-xs text-[#908fa0] font-['Outfit']">
              Standard default schema definitions will be automatically configured for {componentType} components. Additional payload attributes can be assigned upon creation.
            </div>
          )}
        </div>
      </form>
    </Modal>
  );
};

export default CatalogCreationModal;
