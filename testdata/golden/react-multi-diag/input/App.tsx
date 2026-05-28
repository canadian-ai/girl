import React, { useState, useEffect, useCallback, useMemo } from "react";

interface DataItem {
	id: number;
	name: string;
	value: number;
	status: string;
	category: string;
}

interface MetricItem {
	label: string;
	current: number;
	previous: number;
	unit: string;
}

interface DashboardProps {
	title: string;
	userId: number;
	theme: string;
	refreshInterval: number;
}

const sampleData: DataItem[] = [
	{ id: 1, name: "Alpha", value: 42, status: "active", category: "A" },
	{ id: 2, name: "Beta", value: 18, status: "pending", category: "B" },
	{ id: 3, name: "Gamma", value: 75, status: "active", category: "A" },
	{ id: 4, name: "Delta", value: 31, status: "inactive", category: "C" },
	{ id: 5, name: "Epsilon", value: 89, status: "active", category: "B" },
	{ id: 6, name: "Zeta", value: 55, status: "pending", category: "A" },
	{ id: 7, name: "Eta", value: 23, status: "active", category: "C" },
	{ id: 8, name: "Theta", value: 67, status: "inactive", category: "B" },
	{ id: 9, name: "Iota", value: 44, status: "active", category: "A" },
	{ id: 10, name: "Kappa", value: 91, status: "pending", category: "C" },
];

export function Dashboard(props: DashboardProps) {
	const [data, setData] = useState<DataItem[]>(sampleData);
	const [search, setSearch] = useState("");
	const [page, setPage] = useState(1);
	const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
	const [sortField, setSortField] = useState<string>("name");
	const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
	const [statusFilter, setStatusFilter] = useState<string>("all");
	const [categoryFilter, setCategoryFilter] = useState<string>("all");
	const [showFilters, setShowFilters] = useState(false);
	const [isLoading, setIsLoading] = useState(false);

	const filteredData = useMemo(() => {
		let result = data.filter((item) => {
			if (
				props.title &&
				!item.name.toLowerCase().includes(search.toLowerCase())
			) {
				return false;
			}
			if (statusFilter !== "all" && item.status !== statusFilter) {
				return false;
			}
			if (categoryFilter !== "all" && item.category !== categoryFilter) {
				return false;
			}
			return true;
		});
		result.sort((a, b) => {
			const aVal = a[sortField as keyof DataItem];
			const bVal = b[sortField as keyof DataItem];
			if (typeof aVal === "string" && typeof bVal === "string") {
				return sortDir === "asc"
					? aVal.localeCompare(bVal)
					: bVal.localeCompare(aVal);
			}
			if (typeof aVal === "number" && typeof bVal === "number") {
				return sortDir === "asc" ? aVal - bVal : bVal - aVal;
			}
			return 0;
		});
		return result;
	}, [data, search, sortField, sortDir, statusFilter, categoryFilter]);

	const metrics = useMemo((): MetricItem[] => {
		const total = data.length;
		const active = data.filter((d) => d.status === "active").length;
		const pending = data.filter((d) => d.status === "pending").length;
		const avgValue =
			total > 0
				? Math.round(data.reduce((s, d) => s + d.value, 0) / total)
				: 0;
		const categoryCount = new Set(data.map((d) => d.category)).size;
		return [
			{ label: "Total Items", current: total, previous: total - 3, unit: "" },
			{
				label: "Active Items",
				current: active,
				previous: active - 2,
				unit: "",
			},
			{
				label: "Pending Items",
				current: pending,
				previous: pending + 1,
				unit: "",
			},
			{
				label: "Average Value",
				current: avgValue,
				previous: avgValue - 5,
				unit: "pts",
			},
			{
				label: "Categories",
				current: categoryCount,
				previous: categoryCount,
				unit: "",
			},
		];
	}, [data]);

	const handleSearch = useCallback(
		(e: React.ChangeEvent<HTMLInputElement>) => {
			setSearch(e.target.value);
			setPage(1);
		},
		[]
	);

	const handleSort = useCallback(
		(field: string) => {
			if (sortField === field) {
				setSortDir((d) => (d === "asc" ? "desc" : "asc"));
			} else {
				setSortField(field);
				setSortDir("asc");
			}
		},
		[sortField]
	);

	const handleSelect = useCallback((id: number) => {
		setSelectedIds((prev) => {
			const next = new Set(prev);
			if (next.has(id)) {
				next.delete(id);
			} else {
				next.add(id);
			}
			return next;
		});
	}, []);

	const toggleFilters = useCallback(() => {
		setShowFilters((p) => !p);
	}, []);

	useEffect(() => {
		document.title = `${props.title} - Data Dashboard`;
	}, [props.title]);

	useEffect(() => {
		const fetchData = async () => {
			setIsLoading(true);
			try {
				const res = await fetch(`/api/data?userId=${props.userId}`);
				const json: DataItem[] = await res.json();
				if (json.length > 0) {
					setData(json);
				}
			} catch {
				console.error("Failed to fetch data");
			} finally {
				setIsLoading(false);
			}
		};
		fetchData();
	}, [props.userId]);

	useEffect(() => {
		if (props.refreshInterval <= 0) return;
		const timer = setInterval(() => {
			console.log("Auto-refresh dashboard data");
		}, props.refreshInterval);
		return () => clearInterval(timer);
	}, [props.refreshInterval]);

	const totalPages = Math.max(
		1,
		Math.ceil(filteredData.length / 10)
	);
	const pageData = filteredData.slice(
		(page - 1) * 10,
		page * 10
	);

	return (
		<div>
			<div>
				<h1>{props.title}</h1>
				<p>User: {props.userId}</p>
				<p>Theme: {props.theme}</p>
			</div>
			<div>
				<h2>Overview</h2>
				<div>
					<Card>
						<span>{metrics[0].label}</span>
						<span>{metrics[0].current}</span>
					</Card>
					<Card>
						<span>{metrics[1].label}</span>
						<span>{metrics[1].current}</span>
					</Card>
					<Card>
						<span>{metrics[2].label}</span>
						<span>{metrics[2].current}</span>
					</Card>
					<Card>
						<span>{metrics[3].label}</span>
						<span>{metrics[3].current}</span>
					</Card>
					<Card>
						<span>{metrics[4].label}</span>
						<span>{metrics[4].current}</span>
					</Card>
				</div>
			</div>
			<div>
				<div>
					<input
						type="text"
						placeholder="Search items..."
						value={search}
						onChange={handleSearch}
					/>
					<button onClick={toggleFilters}>
						{showFilters ? "Hide" : "Show"} Filters
					</button>
				</div>
				{showFilters && (
					<div>
						<div>
							<label>Status:</label>
							<select
								value={statusFilter}
								onChange={(e) => setStatusFilter(e.target.value)}
							>
								<option value="all">All</option>
								<option value="active">Active</option>
								<option value="pending">Pending</option>
								<option value="inactive">Inactive</option>
							</select>
						</div>
						<div>
							<label>Category:</label>
							<select
								value={categoryFilter}
								onChange={(e) => setCategoryFilter(e.target.value)}
							>
								<option value="all">All</option>
								<option value="A">A</option>
								<option value="B">B</option>
								<option value="C">C</option>
							</select>
						</div>
					</div>
				)}
			</div>
			<div>
				<h2>Data Table</h2>
				{isLoading && <div>Loading...</div>}
				{!isLoading && (
					<Table>
						<thead>
							<tr>
								<th>
									<input
										type="checkbox"
										onChange={() => {
											if (selectedIds.size === pageData.length) {
												setSelectedIds(new Set());
											} else {
												setSelectedIds(
													new Set(pageData.map((d) => d.id))
												);
											}
										}}
										checked={
											pageData.length > 0 &&
											selectedIds.size === pageData.length
										}
									/>
								</th>
								<th onClick={() => handleSort("name")}>Name</th>
								<th onClick={() => handleSort("value")}>Value</th>
								<th onClick={() => handleSort("status")}>Status</th>
								<th onClick={() => handleSort("category")}>Category</th>
							</tr>
						</thead>
						<tbody>
							{pageData.map((item) => (
								<Row key={item.id}>
									<td>
										<input
											type="checkbox"
											checked={selectedIds.has(item.id)}
											onChange={() => handleSelect(item.id)}
										/>
									</td>
									<td>{item.name}</td>
									<td>{item.value}</td>
									<td>{item.status}</td>
									<td>{item.category}</td>
								</Row>
							))}
						</tbody>
					</Table>
				)}
			</div>
			<div>
				{Array.from({ length: totalPages }, (_, i) => i + 1).map(
					(p) => (
						<button
							key={p}
							onClick={() => setPage(p)}
							disabled={p === page}
						>
							{p}
						</button>
					)
				)}
				<span>
					Page {page} of {totalPages}
				</span>
			</div>
			<div>
				<h2>Status Summary</h2>
				<div>
					<Section>
						<span>Active Items</span>
						<span>
							{data.filter((d) => d.status === "active").length}
						</span>
					</Section>
					<Section>
						<span>Pending Items</span>
						<span>
							{data.filter((d) => d.status === "pending").length}
						</span>
					</Section>
					<Section>
						<span>Inactive Items</span>
						<span>
							{data.filter((d) => d.status === "inactive").length}
						</span>
					</Section>
				</div>
			</div>
		</div>
	);
}
