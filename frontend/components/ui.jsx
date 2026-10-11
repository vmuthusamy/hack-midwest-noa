import { ArrowUpRight } from 'lucide-react';
export function Badge({ children, tone = '' }) { return <span className={`badge ${tone}`}>{children}</span>; }
export function Panel({ children, className = '', ...props }) { return <section className={`panel ${className}`} {...props}>{children}</section>; }
export function Button({ children, primary, className = '', ...props }) { return <button className={`button ${primary ? 'primary' : ''} ${className}`} {...props}>{children}</button>; }
export function SourceLink({ href, children }) {
  try {
    const url = new URL(href);
    if (url.protocol !== 'https:' || url.username || url.password || !['api.weather.gov', 'api.waterdata.usgs.gov', 'api.water.noaa.gov'].includes(url.hostname)) return <span>{children}</span>;
  } catch { return <span>{children}</span>; }
  return <a href={href} target="_blank" rel="noreferrer" className="source-link">{children}<ArrowUpRight size={13}/></a>;
}
