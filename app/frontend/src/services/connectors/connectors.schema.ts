export interface ConnectorInfo {
  name: string;
  id: string;
  configuration: Configuration;
  operations: Operation[];
  configs?: string[] | null;
}

// ConnectorConfig is one named config. The api masks password values as "********".
export interface ConnectorConfig {
  id: string;
  connector_id: string;
  name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config: Record<string, any>;
  is_default: boolean;
}

export interface ConnectorConfigInput {
  name?: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config?: Record<string, any>;
  is_default?: boolean;
}

export interface Configuration {
  fields: Field[];
}

export interface Field {
  title: string;
  description: string;
  type: 'text' | 'password' | 'checkbox'; 
  name: string;
  required: boolean;
  editable: boolean;
  visible: boolean;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value?: any; 
  tooltip?: string; 
}

export interface Operation {
  operation: string;
  title: string;
  description: string;
  annotation: string;
  enabled: boolean;
  parameters: Parameter[];
  output?: OutputField[];
}

export interface OutputField {
  name: string;
  type: string;
  description?: string;
}

export interface Parameter {
  title: string;
  description: string;
  required: boolean;
  editable: boolean;
  visible: boolean;
  type: 'text' | 'number' | 'boolean' | 'code' | 'cases' | 'conditions';
  tooltip?: string; 
  name: string; 
  placeholder?: string;
}
