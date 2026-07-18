import React, { useContext, useMemo, useState } from "react";
import { ConnectorInfo } from "@/services/connectors/connectors.schema";
import { PlaybookOperationContext } from "../../../_providers/PlaybookOperationProvider";
import {
  FilterChips,
  Glyph,
  SearchInput,
  connectorGlyph,
} from "@/components/soar";

// Best-effort category bucketing until the backend exposes a category field.
function categoryOf(name: string): string {
  const s = name.toLowerCase();
  if (/(virustotal|threat|sandbox|malware|reputation|firewall|ioc)/.test(s)) return "threat";
  if (/(jira|ticket|servicenow)/.test(s)) return "ticketing";
  if (/(slack|teams|chat|notify|email|mail)/.test(s)) return "communication";
  return "utilities";
}

const CATEGORY_LABELS: Record<string, string> = {
  threat: "Threat intel",
  ticketing: "Ticketing",
  communication: "Communication",
  utilities: "Utilities",
};

const ConnectorList: React.FC<{
  setConnector: React.Dispatch<React.SetStateAction<ConnectorInfo | null>>;
}> = ({ setConnector }) => {
  const { connectorQuery } = useContext(PlaybookOperationContext);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("all");

  const connectors = useMemo(() => connectorQuery?.data ?? [], [connectorQuery?.data]);

  const chips = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const c of connectors) {
      const cat = categoryOf(c.name);
      counts[cat] = (counts[cat] ?? 0) + 1;
    }
    return [
      { value: "all", label: `All (${connectors.length})` },
      ...Object.keys(CATEGORY_LABELS)
        .filter((k) => counts[k])
        .map((k) => ({ value: k, label: `${CATEGORY_LABELS[k]} (${counts[k]})` })),
    ];
  }, [connectors]);

  const filtered = useMemo(
    () =>
      connectors.filter((c) => {
        const matchesSearch = c.name.toLowerCase().includes(search.toLowerCase());
        const matchesCat = category === "all" || categoryOf(c.name) === category;
        return matchesSearch && matchesCat;
      }),
    [connectors, search, category]
  );

  if (connectorQuery == null) return null;

  return (
    <div className="flex flex-1 flex-col gap-3 p-3">
      <SearchInput
        placeholder="Search connectors…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="min-w-0"
      />
      <FilterChips chips={chips} value={category} onChange={setCategory} />

      <div className="flex flex-col gap-2">
        {filtered.map((con) => {
          const g = connectorGlyph(con.name);
          return (
            <button
              key={con.id}
              onClick={() => setConnector(con)}
              className="flex items-start gap-2.5 rounded-md border border-line bg-card p-3 text-left hover:border-line-strong hover:shadow-sm"
            >
              <Glyph icon={g.icon} tone={g.tone} size="md" />
              <div className="min-w-0">
                <div className="text-[13.5px] font-semibold capitalize">{con.name}</div>
                <div className="mt-0.5 text-[12px] text-ink-faint">
                  {CATEGORY_LABELS[categoryOf(con.name)]}
                </div>
                <div className="mt-1 font-mono text-[11.5px] text-ink-faint">
                  {con.operations?.length ?? 0} operations
                </div>
              </div>
            </button>
          );
        })}
        {filtered.length === 0 && (
          <div className="rounded-md border border-dashed border-line-strong px-4 py-6 text-center text-[13px] text-ink-faint">
            No connectors match your search.
          </div>
        )}
      </div>
    </div>
  );
};

export default ConnectorList;
