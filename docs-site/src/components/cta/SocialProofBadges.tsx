// T10 (uiux): Social Proof Badges — badges dinâmicos do GitHub via shields.io.
//
// Sem JS, apenas <img> apontando para shields.io. Cores alinhadas à paleta
// RESMA (labelColor = surface-2 #141519, color = brand/accent/warning/muted).
// loading="lazy" para não bloquear o first paint.

const REPO = 'resma-swarm/resma';
const LABEL_COLOR = '141519'; // --resma-surface-2

type Badge = {
  alt: string;
  src: string;
};

const badges: Badge[] = [
  {
    alt: 'GitHub stars',
    src: `https://img.shields.io/github/stars/${REPO}?style=flat&color=2563eb&labelColor=${LABEL_COLOR}`,
  },
  {
    alt: 'Contributors',
    src: `https://img.shields.io/github/contributors/${REPO}?style=flat&color=3ecf8e&labelColor=${LABEL_COLOR}`,
  },
  {
    alt: 'License',
    src: `https://img.shields.io/github/license/${REPO}?style=flat&color=878787&labelColor=${LABEL_COLOR}`,
  },
  {
    alt: 'Last commit',
    src: `https://img.shields.io/github/last-commit/${REPO}?style=flat&color=f59e0b&labelColor=${LABEL_COLOR}`,
  },
];

export default function SocialProofBadges(): React.JSX.Element {
  return (
    <div className="flex flex-wrap items-center justify-center gap-3">
      {badges.map((b) => (
        <img
          key={b.alt}
          src={b.src}
          alt={b.alt}
          className="h-6"
          loading="lazy"
        />
      ))}
    </div>
  );
}
