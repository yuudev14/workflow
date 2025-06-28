export interface ConnectorInfo {
  name: string;
  id: string;
  configuration: Configuration;
  operations: Operation[];
  configs: string[];
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
}

export interface Parameter {
  title: string;
  description: string;
  required: boolean;
  editable: boolean;
  visible: boolean;
  type: 'text' | 'number' | 'boolean'; 
  tooltip?: string; 
  name?: string; 
  placeholder?: string;
}
