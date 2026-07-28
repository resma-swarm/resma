// T6 (uiux): CodeBlock reutilizavel com syntax highlighting e copy button.
//
// Wrapper sobre prism-react-renderer (ja e dependencia do Docusaurus) com
// tema dracula — mesmo darkTheme configurado em docusaurus.config.ts e ja
// usado em InstallCommand (T4). O fundo do tema e sobrescrito para a surface
// ladder do redesign (canvas -> surface-1 -> surface-2 -> surface-3) com
// hairline borders, mantendo consistencia visual com os demais componentes.
//
// O copy button copia o texto bruto (sem line numbers) para o clipboard e
// mostra feedback de "Copied" por 2s. A logica e identica a de InstallCommand
// (T4), extraida aqui para reuso em outras secoes (ex: T6 Code Example, T7
// Comparison Table).
//
// Referencias:
//   - prism-react-renderer (https://github.com/FormidableLabs/prism-react-renderer)
//   - shadcn Code Block pattern (https://ui.shadcn.com/docs/components/code-block)
//   - Supabase code blocks (https://supabase.com)

import {useState} from 'react';
import {Highlight, themes} from 'prism-react-renderer';
import {Check, Copy} from 'lucide-react';
import clsx from 'clsx';

type CodeBlockProps = {
  /** Codigo bruto a ser exibido (sera trimado antes do highlight). */
  code: string;
  /** Linguagem Prism para highlight (ex: 'http', 'json', 'bash'). */
  language: string;
  /** Titulo exibido no header do bloco (ex: 'request.http'). */
  title?: string;
  /** ClassName extra para o container externo. */
  className?: string;
};

export default function CodeBlock({
  code,
  language,
  title,
  className,
}: CodeBlockProps): React.JSX.Element {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code.trim());
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard pode falhar em contextos sem permissao; silently ignore
    }
  };

  return (
    <div
      className={clsx(
        'rounded-lg border border-hairline bg-surface-2 overflow-hidden',
        className,
      )}>
      {/* Header com titulo + copy button */}
      <div className="flex items-center justify-between gap-2 border-b border-hairline bg-surface-1 px-4 py-2">
        {title ? (
          <span className="font-mono text-xs text-muted">{title}</span>
        ) : (
          <span className="font-mono text-xs text-muted/60">{language}</span>
        )}
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? 'Copied' : 'Copy code'}
          className={clsx(
            'flex items-center justify-center h-7 w-7 rounded-md',
            'border border-hairline bg-surface-3 text-muted',
            'hover:text-ink hover:border-hairline-strong transition-colors',
            copied && 'text-success border-success/40',
          )}>
          {copied ? (
            <Check className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <Copy className="h-3.5 w-3.5" aria-hidden="true" />
          )}
        </button>
      </div>

      {/* Code com syntax highlight + line numbers */}
      <Highlight theme={themes.dracula} code={code.trim()} language={language}>
        {({className: cls, style, tokens, getLineProps, getTokenProps}) => (
          <pre
            className={clsx(
              cls,
              'px-4 py-4 overflow-x-auto font-mono text-sm leading-relaxed',
              // override fundo do tema dracula para surface-2 do redesign
              '!bg-transparent',
            )}
            style={{...style, background: 'transparent'}}>
            {tokens.map((line, i) => {
              const lineProps = getLineProps({line});
              return (
                <div
                  key={i}
                  {...lineProps}
                  className={clsx(lineProps.className, 'table-row')}>
                  <span
                    aria-hidden="true"
                    className="table-cell select-none pr-4 text-right text-muted/60">
                    {i + 1}
                  </span>
                  <span className="table-cell">
                    {line.map((token, key) => {
                      const tokenProps = getTokenProps({token});
                      return <span key={key} {...tokenProps} />;
                    })}
                  </span>
                </div>
              );
            })}
          </pre>
        )}
      </Highlight>
    </div>
  );
}
