import { ArrowUpRight } from 'lucide-react';
export function Badge({ children, tone = '' }) { return <span className={`badge ${tone}`}>{children}</span>; }
export function Panel({ children, className = '', ...props }) { return <section className={`panel ${className}`} {...props}>{children}</section>; }
export function Button({ children, primary, className = '', ...props }) { return <button className={`button ${primary ? 'primary' : ''} ${className}`} {...props}>{children}</button>; }
export function SourceLink({ href, children }) {
  if (!href || !href.startsWith('https://api.weather.gov/')) return <span>{children}</span>;
  return <a href={href} target="_blank" rel="noreferrer" className="source-link">{children}<ArrowUpRight size={13}/></a>;
}
