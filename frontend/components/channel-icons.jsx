import { Bell, BellOff, CheckCheck, CloudLightning, ChevronRight, CircleDashed, CircleMinus, Clock, Forward, Mail, MessageSquareText, Phone, Send, TriangleAlert, UserCheck } from 'lucide-react';
import styles from './channel-icons.module.css';

// In-app uses the Weather Desk mark from the header, drawn as an app tile.
function AppMark() {
  return <span className={styles.appTile} aria-hidden="true"><CloudLightning size={11} strokeWidth={2.5} /></span>;
}

const CHANNELS = {
  'in-app': { Icon: AppMark, label: 'Weather Desk app' },
  email: { Icon: Mail, label: 'Email' },
  sms: { Icon: MessageSquareText, label: 'SMS' },
  voice: { Icon: Phone, label: 'Voice' },
};

// Display the planned order; only recorded simulation events indicate an attempt.
export function ChannelList({ person }) {
  if (!person.included) return <span className={styles.none}><BellOff size={15} aria-hidden="true" />Not targeted</span>;
  const channels = person.channels || [];
  if (!channels.length) return <span className={styles.none}>No channel</span>;
  return <ol className={styles.route} aria-label={`Planned channels in try order: ${channels.map(c => CHANNELS[c]?.label || c).join(', ')}`}>
    {channels.map((c, i) => {
      const { Icon = Bell, label = c } = CHANNELS[c] || {};
      return <li key={c} className={styles.item}>
        {i > 0 && <ChevronRight size={13} className={styles.then} aria-hidden="true" />}
        <span className={styles.chip} data-primary={i === 0 || undefined} title={i === 0 ? `${label} (planned first)` : `${label} (planned fallback)`}>
          <Icon size={15} aria-hidden="true" />{label}
        </span>
      </li>;
    })}
  </ol>;
}

// Status labels come from notificationStatus(); match on the leading word.
const STATES = [
  [/^Not targeted/, CircleMinus, 'muted'],
  [/^Not attempted/, CircleDashed, 'muted'],
  [/^Queued/, Clock, 'info'],
  [/^Attempted/, Send, 'info'],
  [/^Delivered/, CheckCheck, 'good'],
  [/^Acknowledged/, UserCheck, 'good'],
  [/^Unreachable/, TriangleAlert, 'warn'],
  [/^Next channel/, Forward, 'warn'],
];

export function StatusLabel({ label }) {
  const [, Icon, tone] = STATES.find(([re]) => re.test(label)) || [null, CircleDashed, 'muted'];
  return <span className={styles.status} data-tone={tone}><Icon size={15} aria-hidden="true" />{label}</span>;
}
