import { Pipe, PipeTransform } from '@angular/core';
import moment from 'moment';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const RELATIVE_KEYWORD = 'fromNow';
const DEFAULT_EMPTY_PLACEHOLDER = '--';

@Pipe({
  name: 'momentDate',
  standalone: false,
})
export class MomentDatePipe implements PipeTransform {
  /**
   * Formats raw timestamp strings using momentjs.
   * @param value ISO date string or timestamp.
   * @param format Pattern format or 'fromNow'.
   */
  public transform(value: string | Date | undefined | null, format: string = FORMAT_DATETIME_DISPLAY): string {
    if (!value) return DEFAULT_EMPTY_PLACEHOLDER;

    const parsedDate = moment(value);
    if (!parsedDate.isValid()) return DEFAULT_EMPTY_PLACEHOLDER;

    if (format === RELATIVE_KEYWORD) return parsedDate.fromNow();

    return parsedDate.format(format);
  }
}
