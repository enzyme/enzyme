import {
  RadioGroup as AriaRadioGroup,
  Radio as AriaRadio,
  Label,
  type RadioGroupProps as AriaRadioGroupProps,
  type RadioProps as AriaRadioProps,
} from 'react-aria-components';
import { tv } from 'tailwind-variants';

const radioGroup = tv({
  slots: {
    root: 'space-y-2',
    label: 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2',
    radioWrapper: 'flex items-center gap-2',
    radio: [
      'w-4 h-4 rounded-full border-2 transition-colors cursor-pointer',
      'border-gray-300 dark:border-gray-600',
      'focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2',
      'data-[selected]:border-primary-600 data-[selected]:bg-primary-600',
      'after:content-[""] after:block after:w-2 after:h-2 after:rounded-full after:m-auto',
      'data-[selected]:after:bg-white',
    ],
    radioLabel: 'text-sm text-gray-700 dark:text-gray-300 cursor-pointer',
  },
});

interface RadioGroupProps extends Omit<AriaRadioGroupProps, 'className'> {
  label?: string;
  className?: string;
  children: React.ReactNode;
}

export function RadioGroup({ label, className, children, ...props }: RadioGroupProps) {
  const styles = radioGroup();

  return (
    <AriaRadioGroup className={styles.root({ className })} {...props}>
      {label && <Label className={styles.label()}>{label}</Label>}
      <div className="flex gap-4">{children}</div>
    </AriaRadioGroup>
  );
}

interface RadioProps extends Omit<AriaRadioProps, 'className'> {
  children: React.ReactNode;
}

export function Radio({ children, ...props }: RadioProps) {
  const styles = radioGroup();

  return (
    <label className={styles.radioWrapper()}>
      <AriaRadio className={styles.radio()} {...props} />
      <span className={styles.radioLabel()}>{children}</span>
    </label>
  );
}
