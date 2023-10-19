import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { EMPTY, Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { ChatService, ReadbackResponse, ReadbackService, User, VoiceModelResponse } from './chat.module';

@Component({
  selector: 'user-mgmt',
  templateUrl: './user-mgmt.component.html',
})
export class UserManagementComponent implements OnInit {
  // Create a new user.
  createUserName = ''
  createUserStatus = 'normal'

  // User list.
  @ViewChild('refreshButton', { static: true }) refreshButton!: ElementRef;
  userListResp: Observable<User[]> = EMPTY;

  constructor(private chatService: ChatService) {
  }

  ngOnInit() {
    this.userListResp = fromEvent(this.refreshButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.chatService.listUsers()),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  createUserClick() {
    this.chatService.createUser({ Name: this.createUserName, Status: this.createUserStatus } as User).pipe(
      catchError((error) => {
        alert(JSON.stringify(error));
        return EMPTY;
      })
    ).subscribe((result) => {
      alert(JSON.stringify(result));
    });
  }

  refreshUsersClick() {
  }
}
