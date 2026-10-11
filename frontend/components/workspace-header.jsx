import Link from 'next/link';
import { CloudLightning, Moon, Sun } from 'lucide-react';
import { AccountMenu } from './account-menu';
import { Button } from './ui';
export function WorkspaceHeader({identity,theme,onTheme}) {
 return <header className="workspace-header no-print"><Link href="/" className="brand"><CloudLightning size={24}/><span>weather<span className="brand-secondary">desk</span></span></Link><nav aria-label="Workspace"><Link href="/">Manager case</Link><Link href="/incidents">Live weather</Link><Link href="/records">Decision records</Link></nav><div className="workspace-account"><AccountMenu identity={identity}/>{onTheme&&<Button aria-label={`Switch to ${theme==='dark'?'light':'dark'} theme`} onClick={onTheme}>{theme==='dark'?<Sun size={17}/>:<Moon size={17}/>}</Button>}</div></header>;
}
