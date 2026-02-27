import type { SVGProps } from 'react';

export function PinOutlineIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      fill="none"
      aria-hidden="true"
      {...props}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m9 15-6 6M15 6l-1-1 2-2 5 5-2 2-1-1-4.5 4.5c1.5 1.5 1 4-.5 5.5l-8-8c1.5-1.5 4-2 5.5-.5z"
      />
    </svg>
  );
}

export function PinSolidIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <path d="M15.804 2.276a.75.75 0 0 0-.336.195l-2 2a.75.75 0 0 0 0 1.062l.47.469-3.572 3.571c-.83-.534-1.773-.808-2.709-.691-1.183.148-2.32.72-3.187 1.587a.75.75 0 0 0 0 1.063L7.938 15l-5.467 5.467a.75.75 0 0 0 0 1.062.75.75 0 0 0 1.062 0L9 16.062l3.468 3.468a.75.75 0 0 0 1.062 0c.868-.868 1.44-2.004 1.588-3.187.117-.935-.158-1.879-.692-2.708L18 10.063l.469.469a.75.75 0 0 0 1.062 0l2-2a.75.75 0 0 0 0-1.062l-5-4.999a.75.75 0 0 0-.726-.195z" />
    </svg>
  );
}
