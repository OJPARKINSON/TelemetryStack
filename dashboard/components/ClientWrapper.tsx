"use client";

import { useEffect, useState } from "react";

interface ClientWrapperProps {
	children: React.ReactNode;
	fallback?: React.ReactNode;
}

export default function ClientWrapper({
	children,
	fallback,
}: ClientWrapperProps) {
	const [hasMounted, setHasMounted] = useState(false);

	useEffect(() => {
		setHasMounted(true);
	}, []);

	if (!hasMounted) {
		return fallback || null;
	}

	return <>{children}</>;
}
