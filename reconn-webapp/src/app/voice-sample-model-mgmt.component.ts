import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { EMPTY, Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { AIPerson, ChatService, UpdateAIPersonContextPromptByIDParams, VoiceModelResponse, VoiceSample } from './chat.module';

@Component({
  selector: 'voice-sample-model-mgmt',
  templateUrl: './voice-sample-model-mgmt.component.html',
})
export class VoiceSampleModelManagementComponent implements OnInit {
  // Upload voice sample.
  createAIPersonID = '';
  recordingInProgress = false;
  recordButtonCaption = 'Start recording';
  recording?: Blob;
  recordingStatus = new ReplaySubject<string>(1);

  // List voice samples.
  @ViewChild('refreshButton', { static: true }) refreshButton!: ElementRef;
  listAIPersonID = '';
  listResp: Observable<VoiceSample[]> = EMPTY;

  constructor(readonly recorderService: AudioRecorderService, readonly chatService: ChatService) {
    recorderService.recorderError.subscribe((error) => {
      this.recordingStatus.next(JSON.stringify(error));
    });
  }

  ngOnInit() {
    this.listResp = fromEvent(this.refreshButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.chatService.listVoiceSamples(Number(this.listAIPersonID))),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  recordButtonClick() {
    if (this.recordingInProgress) {
      this.recorderService.stopRecording(OutputFormat.BLOB).then((blob) => {
        console.log('recording blob', blob);
        this.recording = blob as Blob;
        this.recordingStatus.next("Recording size: " + this.recording.size / 1024 + "KB");
      });
      this.recordingInProgress = false;
      this.recordButtonCaption = 'Start over';
    } else {
      this.recorderService.startRecording();
      this.recordingInProgress = true;
      this.recordButtonCaption = 'Finish recording';
    }
  }

  uploadButtonClick() {
    if (!this.recording || !this.createAIPersonID) {
      return;
    }
    console.log('input recording', this.recording);
    this.chatService.createVoiceSample(Number(this.createAIPersonID), this.recording).pipe(
      map((resp: VoiceSample) => {
        return resp;
      }),
      catchError((err: HttpErrorResponse) => {
        return of(err);
      })
    ).subscribe((result: unknown) => {
      alert(JSON.stringify(result));
    });
  }
}
