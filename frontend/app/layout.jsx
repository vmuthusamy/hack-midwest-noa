import './globals.css';
export const metadata = { title: 'Weather Desk · Facility operations', description: 'NOAA evidence, reviewed decisions, and simulated notification history.' };
export default function RootLayout({ children }) {
  return <html lang="en" data-scroll-behavior="smooth"><body>{children}</body></html>;
}
