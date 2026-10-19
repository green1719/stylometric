BEGIN;

ALTER TABLE "StylometrySampleTextSamples" RENAME TO "StylometrySamples";
ALTER TABLE "StylometrySampleUsers" RENAME TO "Users";
ALTER TABLE "StylometrySampleLikes" RENAME TO "Likes";

ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleID" TO "ID";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleTextCreator" TO "TextCreator";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleShortDescription" TO "ShortDescription";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleStatus" TO "Status";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleImageURL" TO "ImageURL";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleVideoURL" TO "VideoURL";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleAverageSentenceLength" TO "AverageSentenceLength";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleConjunctionFrequency" TO "ConjunctionFrequency";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleCreatedAt" TO "CreatedAt";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleUpdatedAt" TO "UpdatedAt";
ALTER TABLE "StylometrySamples" RENAME COLUMN "StylometrySampleCreatorID" TO "CreatorID";

ALTER TABLE "Users" RENAME COLUMN "StylometrySampleID" TO "ID";
ALTER TABLE "Users" RENAME COLUMN "StylometrySampleName" TO "Name";
ALTER TABLE "Users" RENAME COLUMN "StylometrySamplePassword" TO "Password";

ALTER TABLE "Likes" RENAME COLUMN "StylometrySampleLikeID" TO "ID";
ALTER TABLE "Likes" RENAME COLUMN "StylometrySampleUserID" TO "UserID";
ALTER TABLE "Likes" RENAME COLUMN "StylometrySampleTextSampleID" TO "SampleID";

ALTER TABLE "StylometrySamples" RENAME CONSTRAINT "StylometrySampleTextSamples_pkey" TO "StylometrySamples_pkey";
ALTER TABLE "Users" RENAME CONSTRAINT "StylometrySampleUsers_pkey" TO "Users_pkey";
ALTER TABLE "Likes" RENAME CONSTRAINT "StylometrySampleLikes_pkey" TO "Likes_pkey";

COMMIT;
