import { redirect } from 'next/navigation';
import WeatherDesk from '../components/weather-desk';
import { currentIdentity } from '../lib/server-identity';
export const dynamic = 'force-dynamic';
export default async function Page({searchParams}) {
  let identity;
  try { identity = await currentIdentity(); } catch { redirect('/signin'); }
  if (!identity) redirect('/signin');
  const query=await searchParams;
  const facility=['barnwell','batesburg','florida','montana','california'].includes(query?.facility)?query.facility:'batesburg';
  return <WeatherDesk key={facility} identity={identity} initialFacility={facility}/>;
}
