import {
  DatePicker as AriaDatePicker,
  DateInput,
  DateSegment,
  Group,
  Button,
  Popover,
  Dialog,
  Calendar,
  CalendarGrid,
  CalendarGridHeader,
  CalendarHeaderCell,
  CalendarGridBody,
  CalendarCell,
  Heading,
  Label,
  type DatePickerProps as AriaDatePickerProps,
  type DateValue,
} from 'react-aria-components';
import { ChevronLeftIcon, ChevronRightIcon, CalendarIcon } from '@heroicons/react/24/outline';
import { tv } from 'tailwind-variants';

const datePicker = tv({
  slots: {
    root: 'group flex flex-col',
    label: 'mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300',
    fieldGroup: [
      'flex items-center rounded-md border border-gray-300 bg-white text-gray-700 shadow-sm',
      'focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500',
      'dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300',
    ],
    input: 'flex flex-1 px-3 py-2',
    segment: [
      'rounded px-0.5 text-end tabular-nums outline-none',
      'focused:bg-blue-500 focused:text-white',
      'placeholder:text-gray-400 dark:placeholder:text-gray-500',
      'data-[type=literal]:px-0',
    ],
    triggerButton: [
      'flex items-center rounded-r px-1.5 text-gray-400 outline-none',
      'hover:text-gray-600 pressed:text-gray-700',
      'dark:hover:text-gray-300 dark:pressed:text-gray-200',
    ],
    popover: [
      'rounded-lg border border-gray-200 bg-white p-3 shadow-lg',
      'dark:border-gray-700 dark:bg-gray-800',
      'entering:animate-in entering:fade-in entering:zoom-in-95 entering:duration-150',
      'exiting:animate-out exiting:fade-out exiting:zoom-out-95 exiting:duration-100',
    ],
    calendarHeader: 'mb-2 flex items-center justify-between',
    calendarHeading: 'text-sm font-semibold text-gray-900 dark:text-gray-100',
    calendarNavButton: [
      'flex h-7 w-7 items-center justify-center rounded-full text-gray-500 outline-none',
      'hover:bg-gray-100 hover:text-gray-700',
      'dark:hover:bg-gray-700 dark:hover:text-gray-300',
      'disabled:text-gray-300 disabled:hover:bg-transparent dark:disabled:text-gray-600',
    ],
    calendarHeaderCell: 'pb-1 text-center text-xs font-medium text-gray-500 dark:text-gray-400',
    calendarCell: [
      'flex h-7 w-7 cursor-pointer items-center justify-center rounded-full text-xs text-gray-700 outline-none',
      'hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700',
      'selected:bg-blue-500 selected:text-white selected:hover:bg-blue-600',
      'disabled:cursor-default disabled:text-gray-300 disabled:hover:bg-transparent dark:disabled:text-gray-600',
      'outside-month:text-gray-300 dark:outside-month:text-gray-600',
      'focused:ring-2 focused:ring-blue-500 focused:ring-offset-1',
    ],
  },
});

interface DatePickerProps<T extends DateValue> extends Omit<AriaDatePickerProps<T>, 'children'> {
  label?: string;
  className?: string;
}

export function DatePicker<T extends DateValue>({
  label,
  className,
  ...props
}: DatePickerProps<T>) {
  const styles = datePicker();

  return (
    <AriaDatePicker {...props} className={styles.root({ className })}>
      {label && <Label className={styles.label()}>{label}</Label>}
      <Group className={styles.fieldGroup()}>
        <DateInput className={styles.input()}>
          {(segment) => <DateSegment segment={segment} className={styles.segment()} />}
        </DateInput>
        <Button className={styles.triggerButton()}>
          <CalendarIcon className="h-4 w-4" />
        </Button>
      </Group>
      <Popover className={styles.popover()} placement="bottom end">
        <Dialog className="outline-none">
          <Calendar>
            <header className={styles.calendarHeader()}>
              <Button slot="previous" className={styles.calendarNavButton()}>
                <ChevronLeftIcon className="h-4 w-4" />
              </Button>
              <Heading className={styles.calendarHeading()} />
              <Button slot="next" className={styles.calendarNavButton()}>
                <ChevronRightIcon className="h-4 w-4" />
              </Button>
            </header>
            <CalendarGrid>
              <CalendarGridHeader>
                {(day) => (
                  <CalendarHeaderCell className={styles.calendarHeaderCell()}>
                    {day}
                  </CalendarHeaderCell>
                )}
              </CalendarGridHeader>
              <CalendarGridBody>
                {(date) => <CalendarCell date={date} className={styles.calendarCell()} />}
              </CalendarGridBody>
            </CalendarGrid>
          </Calendar>
        </Dialog>
      </Popover>
    </AriaDatePicker>
  );
}
