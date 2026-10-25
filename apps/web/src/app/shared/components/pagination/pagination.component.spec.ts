import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PaginationComponent } from './pagination.component';
import { MatIconModule } from '@angular/material/icon';

describe('PaginationComponent', () => {
  let component: PaginationComponent;
  let fixture: ComponentFixture<PaginationComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [PaginationComponent],
      imports: [MatIconModule],
    }).compileComponents();

    fixture = TestBed.createComponent(PaginationComponent);
    component = fixture.componentInstance;
  });

  it('should calculate startItem and endItem correctly', () => {
    component.page = 2;
    component.pageSize = 10;
    component.total = 25;
    component.totalPages = 3;

    expect(component.startItem).toBe(11);
    expect(component.endItem).toBe(20);
    expect(component.isFirstPage).toBe(false);
    expect(component.isLastPage).toBe(false);
  });

  it('should handle zero total items cleanly', () => {
    component.page = 1;
    component.pageSize = 10;
    component.total = 0;
    component.totalPages = 1;

    expect(component.startItem).toBe(0);
    expect(component.endItem).toBe(0);
  });

  it('should emit pageChange when navigating to a valid page', () => {
    component.page = 1;
    component.totalPages = 4;
    const emitSpy = vi.spyOn(component.pageChange, 'emit');

    component.goToNext();
    expect(emitSpy).toHaveBeenCalledWith(2);

    component.page = 2;
    component.goToPrevious();
    expect(emitSpy).toHaveBeenCalledWith(1);
  });

  it('should not emit pageChange when already on the boundary page', () => {
    component.page = 1;
    component.totalPages = 3;
    const emitSpy = vi.spyOn(component.pageChange, 'emit');

    component.goToPrevious();
    expect(emitSpy).not.toHaveBeenCalled();

    component.page = 3;
    component.goToNext();
    expect(emitSpy).not.toHaveBeenCalled();
  });

  it('should emit pageSizeChange on selection', () => {
    const emitSpy = vi.spyOn(component.pageSizeChange, 'emit');
    const mockEvent = {
      target: { value: '20' } as unknown as HTMLSelectElement,
    } as unknown as Event;

    component.onPageSizeSelect(mockEvent);
    expect(emitSpy).toHaveBeenCalledWith(20);
  });
});
