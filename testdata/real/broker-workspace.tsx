import { useState, useCallback, useMemo } from "react";
import { useQuery } from "convex/react";
import { api } from "../convex/_generated/api";

interface BrokerWorkspaceProps {
  brokerId: string;
  workspaceId: string;
}

interface Lead {
  id: string;
  name: string;
  email: string;
  score: number;
  stage: "new" | "contacted" | "qualified" | "proposal" | "closed";
}

interface Campaign {
  id: string;
  name: string;
  leads: Lead[];
  status: "active" | "paused" | "completed";
}

const STAGES = ["new", "contacted", "qualified", "proposal", "closed"] as const;

export default function BrokerWorkspace({
  brokerId,
  workspaceId,
}: BrokerWorkspaceProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterStage, setFilterStage] = useState<string>("all");
  const [selectedLead, setSelectedLead] = useState<Lead | null>(null);

  const workspaceData = useQuery(api.broker.getWorkspace, {
    workspaceId,
    brokerId,
  });

  const leads: Lead[] = useMemo(() => {
    if (!workspaceData?.campaigns) return [];
    return workspaceData.campaigns.flatMap(
      (campaign: Campaign) => campaign.leads
    );
  }, [workspaceData]);

  const filteredLeads = useMemo(() => {
    return leads.filter((lead) => {
      const matchesSearch =
        lead.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        lead.email.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesStage =
        filterStage === "all" || lead.stage === filterStage;
      return matchesSearch && matchesStage;
    });
  }, [leads, searchQuery, filterStage]);

  const stageCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const stage of STAGES) {
      counts[stage] = leads.filter((l) => l.stage === stage).length;
    }
    return counts;
  }, [leads]);

  const handleStageChange = useCallback((leadId: string, stage: string) => {
    console.log("Stage change", leadId, stage);
  }, []);

  const getScoreBadge = (score: number): string => {
    if (score >= 90) return "excellent";
    if (score >= 70) return "good";
    if (score >= 50) return "average";
    return "poor";
  };

  return (
    <div className="broker-workspace">
      <div className="workspace-header">
        <h2>Broker Workspace</h2>
        <div className="search-bar">
          <input
            type="text"
            placeholder="Search leads..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          <select
            value={filterStage}
            onChange={(e) => setFilterStage(e.target.value)}
          >
            <option value="all">All Stages</option>
            {STAGES.map((stage) => (
              <option key={stage} value={stage}>
                {stage.charAt(0).toUpperCase() + stage.slice(1)}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="stage-summary">
        {STAGES.map((stage) => (
          <div key={stage} className="stage-card">
            <span className="stage-name">{stage}</span>
            <span className="stage-count">{stageCounts[stage] || 0}</span>
          </div>
        ))}
      </div>

      <div className="leads-table">
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Score</th>
              <th>Stage</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredLeads.map((lead) => (
              <tr key={lead.id} onClick={() => setSelectedLead(lead)}>
                <td>{lead.name}</td>
                <td>{lead.email}</td>
                <td>
                  <span className={`badge ${getScoreBadge(lead.score)}`}>
                    {lead.score}
                  </span>
                </td>
                <td>{lead.stage}</td>
                <td>
                  <select
                    value={lead.stage}
                    onChange={(e) => handleStageChange(lead.id, e.target.value)}
                  >
                    {STAGES.map((s) => (
                      <option key={s} value={s}>
                        {s}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
