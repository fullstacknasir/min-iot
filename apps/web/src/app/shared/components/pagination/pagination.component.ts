import { Component, Input, Output, EventEmitter } from '@angular/core';
import { PAGE_SIZE_OPTIONS, DEFAULT_PAGE, DEFAULT_PAGE_LIMIT } from '../../../core/constants/app.constants';

const MIN_PAGE_NUMBER = 1;
const PAGE_DELTA_RANGE = 2;

@Component({
  selector: 'app-pagination',
  standalone: false,
  templateUrl: './pagination.component.html',
})
export class PaginationComponent {
  @Input() public page: number = DEFAULT_PAGE;
  @Input() public pageSize: number = DEFAULT_PAGE_LIMIT;
  @Input() public total: number = 0;
  @Input() public totalPages: number = 1;
  @Input() public pageSizeOptions: readonly number[] = PAGE_SIZE_OPTIONS;

  @Output() public readonly pageChange = new EventEmitter<number>();
  @Output() public readonly pageSizeChange = new EventEmitter<number>();

  public get startItem(): number {
    if (this.total === 0) return 0;
    return (this.page - 1) * this.pageSize + 1;
  }

  public get endItem(): number {
    return Math.min(this.page * this.pageSize, this.total);
  }

  public get isFirstPage(): boolean {
    return this.page <= MIN_PAGE_NUMBER;
  }

  public get isLastPage(): boolean {
    return this.page >= this.totalPages || this.totalPages <= MIN_PAGE_NUMBER;
  }

  /**
   * Generates a sliding window of visible page numbers around the current page.
   */
  public getVisiblePages(): number[] {
    const total = Math.max(MIN_PAGE_NUMBER, this.totalPages);
    const current = Math.min(Math.max(MIN_PAGE_NUMBER, this.page), total);
    const start = Math.max(MIN_PAGE_NUMBER, current - PAGE_DELTA_RANGE);
    const end = Math.min(total, current + PAGE_DELTA_RANGE);

    return Array.from({ length: end - start + 1 }, (_, index) => start + index);
  }

  public goToPage(targetPage: number): void {
    if (targetPage < MIN_PAGE_NUMBER || targetPage > this.totalPages || targetPage === this.page) {
      return;
    }
    this.pageChange.emit(targetPage);
  }

  public goToPrevious(): void {
    if (this.isFirstPage) return;
    this.goToPage(this.page - 1);
  }

  public goToNext(): void {
    if (this.isLastPage) return;
    this.goToPage(this.page + 1);
  }

  public onPageSizeSelect(event: Event): void {
    const selectElement = event.target as HTMLSelectElement;
    const newSize = Number(selectElement?.value);
    if (!newSize || newSize === this.pageSize) return;
    this.pageSizeChange.emit(newSize);
  }
}
