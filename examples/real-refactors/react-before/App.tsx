import React, { useState, useEffect, useCallback } from "react";

const MOCK_USERS = [
  { id: "1", name: "Alice Wang", email: "alice@example.com", role: "admin", status: "active", department: "engineering", lastLogin: "2026-05-26", avatar: null },
  { id: "2", name: "Bob Smith", email: "bob@example.com", role: "editor", status: "active", department: "design", lastLogin: "2026-05-25", avatar: null },
  { id: "3", name: "Carol Jones", email: "carol@example.com", role: "viewer", status: "inactive", department: "marketing", lastLogin: "2026-04-30", avatar: null },
  { id: "4", name: "Dave Patel", email: "dave@example.com", role: "editor", status: "active", department: "engineering", lastLogin: "2026-05-26", avatar: null },
  { id: "5", name: "Eve Kim", email: "eve@example.com", role: "admin", status: "active", department: "operations", lastLogin: "2026-05-24", avatar: null },
  { id: "6", name: "Frank Lee", email: "frank@example.com", role: "viewer", status: "suspended", department: "design", lastLogin: "2026-03-15", avatar: null },
  { id: "7", name: "Grace Chen", email: "grace@example.com", role: "editor", status: "active", department: "marketing", lastLogin: "2026-05-26", avatar: null },
  { id: "8", name: "Henry Brown", email: "henry@example.com", role: "viewer", status: "inactive", department: "engineering", lastLogin: "2026-05-01", avatar: null },
];

const DEPARTMENTS = ["engineering", "design", "marketing", "operations", "sales"];
const ROLES = ["admin", "editor", "viewer"];
const STATUSES = ["active", "inactive", "suspended"];

export default function UserDashboard() {
  const [users, setUsers] = useState<typeof MOCK_USERS>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [filterRole, setFilterRole] = useState("all");
  const [filterDepartment, setFilterDepartment] = useState("all");
  const [filterStatus, setFilterStatus] = useState("all");
  const [sortField, setSortField] = useState<"name" | "role" | "lastLogin">("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [selectedUserIds, setSelectedUserIds] = useState<Set<string>>(new Set());
  const [showFilters, setShowFilters] = useState(false);
  const [isFormOpen, setIsFormOpen] = useState(false);

  useEffect(() => {
    const fetchUsers = async () => {
      setLoading(true);
      setError(null);
      try {
        await new Promise((r) => setTimeout(r, 500));
        setUsers(MOCK_USERS);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to fetch users");
      } finally {
        setLoading(false);
      }
    };
    fetchUsers();
  }, []);

  useEffect(() => {
    setSelectedUserIds(new Set());
  }, [filterRole, filterDepartment, filterStatus, searchQuery]);

  useEffect(() => {
    console.log("UserDashboard rendered with", users.length, "users");
  });

  const handleSort = useCallback((field: typeof sortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
        return prev;
      }
      setSortDirection("asc");
      return field;
    });
  }, []);

  const toggleSelectUser = useCallback((id: string) => {
    setSelectedUserIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    setSelectedUserIds((prev) => {
      if (prev.size === filteredUsers.length) return new Set();
      return new Set(filteredUsers.map((u) => u.id));
    });
  }, [filteredUsers]);

  const filteredUsers = users
    .filter((u) => {
      const q = searchQuery.toLowerCase();
      return (
        u.name.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q)
      );
    })
    .filter((u) => filterRole === "all" || u.role === filterRole)
    .filter((u) => filterDepartment === "all" || u.department === filterDepartment)
    .filter((u) => filterStatus === "all" || u.status === filterStatus)
    .sort((a, b) => {
      const dir = sortDirection === "asc" ? 1 : -1;
      if (sortField === "name") return a.name.localeCompare(b.name) * dir;
      if (sortField === "role") return a.role.localeCompare(b.role) * dir;
      return a.lastLogin.localeCompare(b.lastLogin) * dir;
    });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Loading users...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-4xl mx-auto p-4">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          Error: {error}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto p-4">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">User Dashboard</h1>
        <button
          onClick={() => setIsFormOpen(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        >
          Add User
        </button>
      </div>

      <div className="mb-4">
        <input
          type="text"
          placeholder="Search by name or email..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full border rounded px-3 py-2"
        />
      </div>

      <button
        onClick={() => setShowFilters(!showFilters)}
        className="text-sm text-blue-600 hover:text-blue-800 mb-4"
      >
        {showFilters ? "Hide" : "Show"} Filters
      </button>

      {showFilters && (
        <div className="flex gap-4 mb-4 p-4 bg-gray-50 rounded">
          <div>
            <label className="block text-xs font-medium mb-1">Role</label>
            <select
              value={filterRole}
              onChange={(e) => setFilterRole(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            >
              <option value="all">All Roles</option>
              {ROLES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium mb-1">Department</label>
            <select
              value={filterDepartment}
              onChange={(e) => setFilterDepartment(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            >
              <option value="all">All Departments</option>
              {DEPARTMENTS.map((d) => (
                <option key={d} value={d}>{d}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium mb-1">Status</label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            >
              <option value="all">All Statuses</option>
              {STATUSES.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>
        </div>
      )}

      <div className="mb-2 text-sm text-gray-500">
        {filteredUsers.length} of {users.length} users
        {selectedUserIds.size > 0 && ` (${selectedUserIds.size} selected)`}
      </div>

      <div className="border rounded">
        <div className="flex items-center gap-4 px-4 py-2 bg-gray-50 border-b text-sm font-medium text-gray-600">
          <div className="w-8">
            <input
              type="checkbox"
              checked={selectedUserIds.size === filteredUsers.length && filteredUsers.length > 0}
              onChange={toggleSelectAll}
            />
          </div>
          <button className="flex-1 text-left" onClick={() => handleSort("name")}>
            Name {sortField === "name" ? (sortDirection === "asc" ? "\u2191" : "\u2193") : ""}
          </button>
          <div className="flex-1">Email</div>
          <button className="w-24 text-left" onClick={() => handleSort("role")}>
            Role {sortField === "role" ? (sortDirection === "asc" ? "\u2191" : "\u2193") : ""}
          </button>
          <div className="w-28">Department</div>
          <div className="w-20">Status</div>
          <button className="w-28 text-left" onClick={() => handleSort("lastLogin")}>
            Last Login {sortField === "lastLogin" ? (sortDirection === "asc" ? "\u2191" : "\u2193") : ""}
          </button>
          <div className="w-20">Actions</div>
        </div>

        {filteredUsers.map((user) => (
          <div
            key={user.id}
            className={`flex items-center gap-4 px-4 py-3 border-b last:border-0 hover:bg-gray-50 ${
              selectedUserIds.has(user.id) ? "bg-blue-50" : ""
            }`}
          >
            <div className="w-8">
              <input
                type="checkbox"
                checked={selectedUserIds.has(user.id)}
                onChange={() => toggleSelectUser(user.id)}
              />
            </div>
            <div className="flex-1 font-medium">{user.name}</div>
            <div className="flex-1 text-sm text-gray-600">{user.email}</div>
            <div className="w-24 text-sm">{user.role}</div>
            <div className="w-28 text-sm text-gray-600">{user.department}</div>
            <div className="w-20">
              <span
                className={`inline-block px-2 py-0.5 text-xs rounded-full ${
                  user.status === "active"
                    ? "bg-green-100 text-green-800"
                    : user.status === "inactive"
                    ? "bg-yellow-100 text-yellow-800"
                    : "bg-red-100 text-red-800"
                }`}
              >
                {user.status}
              </span>
            </div>
            <div className="w-28 text-xs text-gray-500">{user.lastLogin}</div>
            <div className="w-20">
              <button
                onClick={() => console.log("edit", user.id)}
                className="text-blue-600 hover:text-blue-800 text-sm mr-2"
              >
                Edit
              </button>
              <button
                onClick={() => {
                  if (confirm("Delete user?")) {
                    setUsers((prev) => prev.filter((u) => u.id !== user.id));
                  }
                }}
                className="text-red-600 hover:text-red-800 text-sm"
              >
                Delete
              </button>
            </div>
          </div>
        ))}

        {filteredUsers.length === 0 && (
          <div className="px-4 py-8 text-center text-gray-500">
            No users found matching your criteria.
          </div>
        )}
      </div>

      {isFormOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">Add User</h2>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Name</label>
                <input type="text" className="w-full border rounded px-3 py-2" placeholder="Enter name" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Email</label>
                <input type="email" className="w-full border rounded px-3 py-2" placeholder="Enter email" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Role</label>
                <select className="w-full border rounded px-3 py-2">
                  {ROLES.map((r) => (
                    <option key={r} value={r}>{r}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Department</label>
                <select className="w-full border rounded px-3 py-2">
                  {DEPARTMENTS.map((d) => (
                    <option key={d} value={d}>{d}</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => setIsFormOpen(false)}
                className="px-4 py-2 text-sm border rounded hover:bg-gray-50"
              >
                Cancel
              </button>
              <button className="px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
