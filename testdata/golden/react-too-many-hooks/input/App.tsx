import React, { useState, useEffect, useCallback, useMemo, useRef, useLayoutEffect } from "react";

export function Dashboard() {
	const [a, setA] = useState(0);
	const [b, setB] = useState("");
	const [c, setC] = useState(false);
	const d = useRef(null);
	const e = useMemo(() => a, [a]);
	const f = useCallback(() => {}, []);
	useEffect(() => {
		document.title = String(a);
	}, [a]);
	useLayoutEffect(() => {
		console.log(b);
	}, [b]);

	return <div>{a}{b}</div>;
}
