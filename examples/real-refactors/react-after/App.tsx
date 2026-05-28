import React from "react";
import { UserList } from "./UserList";
import { SearchBar } from "./SearchBar";
import { FilterBar } from "./FilterBar";
import { AddUserModal } from "./AddUserModal";
import { useUsers } from "./hooks/useUsers";
import { useSearch } from "./hooks/useSearch";

export default function UserDashboard() {
  const {
    users,
    loading,
    error,
    selectedUserIds,
    toggleSelectUser,
    toggleSelectAll,
    deleteUser,
  } = useUsers();

  const {
    searchQuery,
    setSearchQuery,
    filterRole,
    setFilterRole,
    filterDepartment,
    setFilterDepartment,
    filterStatus,
    setFilterStatus,
    showFilters,
    setShowFilters,
    sortField,
    sortDirection,
    handleSort,
    filteredUsers,
  } = useSearch(users);

  const [isFormOpen, setIsFormOpen] = React.useState(false);

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

  const allSelected =
    filteredUsers.length > 0 &&
    selectedUserIds.size === filteredUsers.length;

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

      <SearchBar value={searchQuery} onChange={setSearchQuery} />

      <FilterBar
        show={showFilters}
        onToggle={() => setShowFilters(!showFilters)}
        filterRole={filterRole}
        onFilterRoleChange={setFilterRole}
        filterDepartment={filterDepartment}
        onFilterDepartmentChange={setFilterDepartment}
        filterStatus={filterStatus}
        onFilterStatusChange={setFilterStatus}
      />

      <UserList
        users={filteredUsers}
        totalCount={users.length}
        selectedUserIds={selectedUserIds}
        allSelected={allSelected}
        sortField={sortField}
        sortDirection={sortDirection}
        onSort={handleSort}
        onToggleSelect={toggleSelectUser}
        onToggleSelectAll={toggleSelectAll}
        onDeleteUser={deleteUser}
      />

      {isFormOpen && <AddUserModal onClose={() => setIsFormOpen(false)} />}
    </div>
  );
}
