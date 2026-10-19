import { MomentDatePipe } from './moment-date.pipe';
import moment from 'moment';

describe('MomentDatePipe', () => {
  let pipe: MomentDatePipe;

  beforeEach(() => {
    pipe = new MomentDatePipe();
  });

  it('should return placeholder for null or undefined dates', () => {
    expect(pipe.transform(null)).toBe('--');
    expect(pipe.transform(undefined)).toBe('--');
    expect(pipe.transform('')).toBe('--');
  });

  it('should format ISO timestamp with default format', () => {
    const isoString = '2026-10-19T10:00:00Z';
    const result = pipe.transform(isoString);
    expect(result).toBe(moment(isoString).format('YYYY-MM-DD HH:mm:ss'));
  });

  it('should support fromNow relative format', () => {
    const recentDate = moment().subtract(5, 'minutes').toISOString();
    const result = pipe.transform(recentDate, 'fromNow');
    expect(result).toContain('ago');
  });
});
