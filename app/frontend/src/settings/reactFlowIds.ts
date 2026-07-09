export const FLOW_START_ID = "start"
export const FLOW_SELECT_TRIGGER_ID = "select_trigger"

// the condition builtin: each outgoing edge carries a source_handle naming its
// branch ("true"/"false", a switch case id, or "else"), read by the executor.
export const CONDITION_CONNECTOR_ID = "condition"

// virtual "code snippet" connectors — only info.json exists in the tree, the
// sandbox harness IS the implementation. Single-operation, no config.
export const CODE_SNIPPET_PY_ID = "code_snippet"
export const CODE_SNIPPET_JS_ID = "code_snippet_js"

// default handle for an unrouted condition edge. Non-directional and never a
// branch id, so the executor skips it until the editor assigns a branch.
export const CONDITION_OUTPUT_HANDLE = "output"