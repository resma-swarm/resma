// T2 (uiux): Badge stack acima do H1 do hero.
//
// Três badges hardcoded (MIT License, Self-hosted, 0 stars) com ícones Lucide.
// Badges dinâmicas do GitHub ficam para T10 — aqui mantemos estático para evitar
// fetch em runtime no landing.

import {Shield, Server, Star} from 'lucide-react';
import type {LucideIcon} from 'lucide-react';

type Badge = {
  icon: LucideIcon;
  label: string;
};

const badges: Badge[] = [
  {icon: Shield, label: 'MIT License'},
  {icon: Server, label: 'Self-hosted'},
  {icon: Star, label: '0 stars'},
];

export default function BadgeStack(): React.JSX.Element {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {badges.map(({icon: Icon, label}) => (
        <span
          key={label}
          className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs bg-surface-2 border border-hairline text-muted">
          <Icon className="h-3.5 w-3.5" aria-hidden="true" />
          {label}
        </span>
      ))}
    </div>
  );
}
