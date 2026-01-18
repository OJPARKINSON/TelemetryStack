import { ThemeProvider } from "../../components/theme-provider";
import "./globals.css";

// Conditionally import Service Worker only in production
const ServiceWorkerProvider =
	import.meta.env.MODE === "production"
		? require("../components/ServiceWorkerProvider").default
		: () => null;

export default function RootLayout({
	children,
}: Readonly<{
	children: React.ReactNode;
}>) {
	return (
		<html lang="en" suppressHydrationWarning>
			<body>
				{import.meta.env.MODE === "production" && <ServiceWorkerProvider />}
				<ThemeProvider attribute="class" defaultTheme="system" enableSystem>
					{children}
				</ThemeProvider>
			</body>
		</html>
	);
}
