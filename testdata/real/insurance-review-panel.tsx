import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation } from "convex/react";
import { api } from "../convex/_generated/api";

interface InsuranceReviewPanelProps {
  tenantId: string;
  workspaceId: string;
  onComplete: (reviewId: string) => void;
}

interface DocumentItem {
  id: string;
  name: string;
  status: "pending" | "approved" | "rejected";
  url: string;
}

interface ReviewState {
  documents: DocumentItem[];
  readiness: number;
  assignedBrokerId: string | null;
}

const DEFAULT_REVIEW: ReviewState = {
  documents: [],
  readiness: 0,
  assignedBrokerId: null,
};

export default function InsuranceReviewPanel({
  tenantId,
  workspaceId,
  onComplete,
}: InsuranceReviewPanelProps) {
  const [review, setReview] = useState<ReviewState>(DEFAULT_REVIEW);
  const [selectedDoc, setSelectedDoc] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const tenantData = useQuery(api.tenants.getById, { id: tenantId });
  const workspaceDocs = useQuery(api.workspaces.getDocuments, {
    workspaceId,
  });
  const submitReview = useMutation(api.reviews.submit);

  useEffect(() => {
    if (workspaceDocs) {
      setReview((prev) => ({
        ...prev,
        documents: workspaceDocs.map((doc: any) => ({
          id: doc._id,
          name: doc.name,
          status: doc.status,
          url: doc.url,
        })),
      }));
    }
  }, [workspaceDocs]);

  useEffect(() => {
    if (review.documents.length > 0) {
      const approved = review.documents.filter(
        (d) => d.status === "approved"
      ).length;
      const readiness = Math.round((approved / review.documents.length) * 100);
      setReview((prev) => ({ ...prev, readiness }));
    }
  }, [review.documents]);

  const handleStatusChange = useCallback(
    async (docId: string, status: "approved" | "rejected") => {
      setReview((prev) => ({
        ...prev,
        documents: prev.documents.map((d) =>
          d.id === docId ? { ...d, status } : d
        ),
      }));
    },
    []
  );

  const handleSubmit = useCallback(async () => {
    setIsSubmitting(true);
    try {
      const result = await submitReview({
        tenantId,
        workspaceId,
        documents: review.documents,
        readiness: review.readiness,
      });
      onComplete(result._id);
    } finally {
      setIsSubmitting(false);
    }
  }, [tenantId, workspaceId, review, submitReview, onComplete]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && selectedDoc) {
        handleStatusChange(selectedDoc, "approved");
      }
      if (e.key === "Escape") {
        setSelectedDoc(null);
      }
    },
    [selectedDoc, handleStatusChange]
  );

  const getReadinessColor = (score: number): string => {
    if (score >= 80) return "green";
    if (score >= 50) return "yellow";
    return "red";
  };

  return (
    <div className="insurance-panel" onKeyDown={handleKeyDown}>
      <h2>Insurance Review</h2>

      {review.documents.length === 0 ? (
        <div className="empty-state">No documents to review</div>
      ) : (
        <div className="document-list">
          {review.documents.map((doc) => (
            <div
              key={doc.id}
              className={`document-item ${doc.status}`}
              onClick={() => setSelectedDoc(doc.id)}
            >
              <span className="doc-name">{doc.name}</span>
              <span className={`doc-status ${doc.status}`}>{doc.status}</span>
            </div>
          ))}
        </div>
      )}

      <div className="readiness-section">
        <label>Readiness Score</label>
        <div
          className="readiness-bar"
          style={{ backgroundColor: getReadinessColor(review.readiness) }}
        >
          <div
            className="readiness-fill"
            style={{ width: `${review.readiness}%` }}
          />
        </div>
        <span className="readiness-value">{review.readiness}%</span>
      </div>

      <div className="actions">
        <button onClick={handleSubmit} disabled={isSubmitting}>
          {isSubmitting ? "Submitting..." : "Submit Review"}
        </button>
      </div>

      {selectedDoc && (
        <div className="modal">
          <div className="modal-content">
            <h3>Review Document</h3>
            <button onClick={() => handleStatusChange(selectedDoc, "approved")}>
              Approve
            </button>
            <button onClick={() => handleStatusChange(selectedDoc, "rejected")}>
              Reject
            </button>
            <button onClick={() => setSelectedDoc(null)}>Close</button>
          </div>
        </div>
      )}
    </div>
  );
}
