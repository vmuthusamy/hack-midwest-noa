'use client';
import { signOut } from 'next-auth/react';
import { LogOut } from 'lucide-react';
export function AccountMenu({ identity }) {
  if (identity.demo) return <span className="prototype-label">DEMO MANAGER · NOT SIGNED IN</span>;
  return <div className="account-menu"><span><strong>{identity.name}</strong><small>{identity.role === 'manager' ? 'Manager' : 'Viewer'} · {identity.email || identity.login}</small></span><button className="button" onClick={()=>signOut({ redirectTo: '/signin' })} aria-label="Sign out"><LogOut size={16}/><span>Sign out</span></button></div>;
}
