import { redirect } from 'next/navigation';
import WeatherDesk from '../components/weather-desk';
import { currentIdentity } from '../lib/server-identity';
export const dynamic = 'force-dynamic';
export default async function Page() {
  let identity;
  try { identity = await currentIdentity(); } catch { redirect('/signin'); }
  if (!identity) redirect('/signin');
  return <WeatherDesk identity={identity}/>;
}
