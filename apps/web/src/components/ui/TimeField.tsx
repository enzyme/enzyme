import {
  TimeField as AriaTimeField,
  DateInput,
  DateSegment,
  Label,
  type TimeFieldProps as AriaTimeFieldProps,
  type TimeValue,
} from 'react-aria-components';
import { tv } from 'tailwind-variants';

const timeField = tv({
  slots: {
    root: 'group flex flex-col',
    label: 'mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300',
    input: [
      'flex items-center rounded-md border border-gray-300 bg-white px-3 py-2 text-gray-700 shadow-sm',
      'focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500',
      'dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300',
    ],
    segment: [
      'rounded px-0.5 text-end tabular-nums outline-none',
      'focused:bg-blue-500 focused:text-white',
      'placeholder:text-gray-400 dark:placeholder:text-gray-500',
      'data-[type=literal]:px-0',
    ],
  },
});

interface TimeFieldProps<T extends TimeValue> extends Omit<AriaTimeFieldProps<T>, 'children'> {
  label?: string;
  className?: string;
}

export function TimeField<T extends TimeValue>({ label, className, ...props }: TimeFieldProps<T>) {
  const styles = timeField();

  return (
    <AriaTimeField {...props} className={styles.root({ className })}>
      {label && <Label className={styles.label()}>{label}</Label>}
      <DateInput className={styles.input()}>
        {(segment) => <DateSegment segment={segment} className={styles.segment()} />}
      </DateInput>
    </AriaTimeField>
  );
}
